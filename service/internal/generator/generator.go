package generator

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/jamesread/StencilBox/internal/buildconfigs"
	"github.com/jamesread/StencilBox/internal/buildinfo"
	"github.com/jamesread/StencilBox/internal/scraper"
	"github.com/jamesread/golure/pkg/dirs"
	"github.com/jamesread/golure/pkg/easyexec"
	"github.com/jamesread/golure/pkg/git"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// totalBuildTimeout is the maximum wall-clock time for a single generate run (template, favicons, git, vite).
const totalBuildTimeout = 10 * time.Minute

type BuildStatus struct {
	Message                 string
	BuildUrlBase            string
	OutputSizeHumanReadable string
	IsError                 bool
}

type BuildContext struct {
	Ctx           context.Context
	BuildConfig   *buildconfigs.BuildConfig
	BuildStatus   *BuildStatus
	UpdateChannel chan string
	TemplateData  map[string]any
}

func Generate(parentCtx context.Context, baseOutputDir string, cfg *buildconfigs.BuildConfig, buildStatus *BuildStatus, updateChannel chan string) {
	if parentCtx == nil {
		parentCtx = context.Background()
	}
	deadlineCtx, cancel := context.WithTimeout(parentCtx, totalBuildTimeout)
	defer cancel()

	bctx := &BuildContext{
		Ctx:           deadlineCtx,
		BuildConfig:   cfg,
		BuildStatus:   buildStatus,
		UpdateChannel: updateChannel,
		TemplateData:  make(map[string]any),
	}

	updateChannel <- fmt.Sprintf("Starting build for project %s", cfg.Name)

	defer close(updateChannel)

	run := &buildRun{BuildContext: bctx, baseOutputDir: baseOutputDir}
	run.execute()
}

// buildRun holds the state shared between the steps of a single build.
type buildRun struct {
	*BuildContext

	tmpl           *template.Template
	outfile        *os.File
	templateDir    TemplateDir
	baseOutputDir  string
	finalOutputDir string
	tempOutputDir  string
	repoStorageDir string
}

// execute runs each build step in order. A step returns false when the build must stop, after
// recording why in the build status.
func (r *buildRun) execute() {
	defer r.closeOutput()

	steps := []func() bool{
		r.prepareDirs,
		r.loadTemplate,
		r.loadDatafiles,
		r.processLinks,
		r.openOutputFile,
		r.cloneRepositories,
		r.renderTemplate,
		r.copyLayerFiles,
		r.runViteBuild,
	}

	for _, step := range steps {
		if !step() {
			return
		}
	}

	r.finish()
}

func (r *buildRun) fail(message string) bool {
	failBuild(r.BuildStatus, r.UpdateChannel, message)
	return false
}

// running reports whether the build may continue (it has not been cancelled or timed out).
func (r *buildRun) running() bool {
	return !abortBuild(r.Ctx, r.BuildStatus, r.UpdateChannel)
}

func (r *buildRun) closeOutput() {
	if r.outfile != nil {
		_ = r.outfile.Close()
	}
}

func (r *buildRun) prepareDirs() bool {
	cfg := r.BuildConfig
	r.finalOutputDir = filepath.Join(r.baseOutputDir, cfg.OutputDir)

	tempDir, err := TemporaryOutputDir(r.baseOutputDir, cfg.OutputDir)
	if err != nil {
		return r.fail("Invalid output directory: " + err.Error())
	}
	r.tempOutputDir = tempDir

	log.WithFields(log.Fields{
		"name":           cfg.Name,
		"outputDir":      cfg.OutputDir,
		"finalOutputDir": r.finalOutputDir,
	}).Infof("Starting build for project")

	if err = os.MkdirAll(r.finalOutputDir, 0o755); err != nil {
		return r.fail("Failed to create output directory: " + err.Error())
	}
	if err = os.MkdirAll(r.tempOutputDir, 0o755); err != nil {
		return r.fail("Failed to create temporary output directory: " + err.Error())
	}

	return true
}

func (r *buildRun) loadTemplate() bool {
	templateDir, err := FindTemplate(r.BuildConfig.Template)
	if err != nil {
		return r.fail("Failed to find template: " + err.Error())
	}
	r.templateDir = templateDir

	r.UpdateChannel <- fmt.Sprintf("Using %s template from %s", templateDir.Source, templateDir.Path)

	indexPath := filepath.Join(templateDir.Path, "index.html")

	r.tmpl, err = getNewTemplater(r.tempOutputDir).ParseFiles(indexPath)
	if err != nil {
		return r.fail("Failed to parse template: " + err.Error())
	}

	return r.running()
}

func (r *buildRun) loadDatafiles() bool {
	for name, path := range r.BuildConfig.Datafiles {
		log.Infof("Adding datafile %v with path %v", name, path)

		dat, err := readDatafile(path, filepath.Dir(r.BuildConfig.Path))
		if err != nil {
			return r.fail("Error reading datafile " + name + ": " + err.Error())
		}

		if dat == nil {
			continue
		}

		r.TemplateData[name] = dat
	}

	log.Infof("Datafiles loaded: %+v", r.TemplateData)

	return r.running()
}

// processLinks adds favicons to links data.
func (r *buildRun) processLinks() bool {
	linksData, ok := r.TemplateData["links"]
	if !ok {
		return r.running()
	}

	r.UpdateChannel <- "Fetching favicons for links..."
	processedLinks, err := processLinksWithFavicons(r.Ctx, linksData, r.tempOutputDir, r.UpdateChannel)
	if err == nil {
		r.TemplateData["links"] = processedLinks
		return r.running()
	}

	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		abortBuild(r.Ctx, r.BuildStatus, r.UpdateChannel)
		return false
	}

	// Continue with original data if favicon processing fails
	log.Warnf("Failed to process links with favicons: %v", err)

	return r.running()
}

func (r *buildRun) openOutputFile() bool {
	outfile, err := os.Create(r.tempOutputDir + "/index.html")
	if err != nil {
		log.Errorf("Error creating output file: %v", err)
		return r.fail("Error creating output file: " + err.Error())
	}
	r.outfile = outfile

	return true
}

func (r *buildRun) cloneRepositories() bool {
	r.repoStorageDir = getRepoStorageDir(r.BuildConfig)
	cloneRepos(r.BuildContext, r.repoStorageDir)

	return r.running()
}

func (r *buildRun) addTemplateGlobals() {
	cfg := r.BuildConfig

	r.TemplateData["hooks"] = buildRepoHooks(cfg, r.UpdateChannel)
	r.TemplateData["buildconfig"] = cfg

	// Add build date/time and StencilBox version to template data
	buildTime := time.Now()
	r.TemplateData["buildDate"] = buildTime.Format(time.RFC3339)
	r.TemplateData["buildDateFormatted"] = buildTime.Format("January 2, 2006 at 3:04 PM MST")
	r.TemplateData["version"] = buildinfo.Version
	r.TemplateData["webuiUrl"] = webuiBuildConfigURL(getWebuiUrlBase(), cfg.Name)
}

func (r *buildRun) renderTemplate() bool {
	r.addTemplateGlobals()

	log.Infof("Template data: %+v", r.TemplateData)

	if err := r.tmpl.Execute(r.outfile, r.TemplateData); err != nil {
		return r.fail("Error executing template: " + err.Error())
	}

	if err := r.outfile.Close(); err != nil {
		return r.fail("Error writing output file: " + err.Error())
	}

	return r.running()
}

func (r *buildRun) copyLayerFiles() bool {
	copyLayers(r.tempOutputDir)
	if err := copyTemplateLayers(r.templateDir.Path, r.tempOutputDir); err != nil {
		return r.fail("Failed to copy template layers: " + err.Error())
	}

	lnRepos(r.tempOutputDir, r.repoStorageDir)

	return r.running()
}

func (r *buildRun) runViteBuild() bool {
	r.UpdateChannel <- "Running Vite build..."

	runVite(r.BuildContext, r.tempOutputDir, r.finalOutputDir)

	return r.running() && !r.BuildStatus.IsError
}

func (r *buildRun) finish() {
	r.UpdateChannel <- "Build completed!"

	r.BuildStatus.Message = "Build completed successfully"
	r.BuildStatus.IsError = false
	r.BuildStatus.BuildUrlBase = getBuildUrlBase()
	r.BuildStatus.OutputSizeHumanReadable = getDirectorySizeHumanReadable(r.finalOutputDir)
}

func buildRepoHooks(cfg *buildconfigs.BuildConfig, updateChannel chan string) *map[string]string {
	updateChannel <- "Building repo hooks..."

	hooks := make(map[string]string)

	for _, repo := range cfg.Repos {
		appendHookData("head", &hooks, &repo, cfg)
		appendHookData("body", &hooks, &repo, cfg)
	}

	return &hooks
}

func appendHookData(hookName string, hooks *map[string]string, repo *buildconfigs.GitRepo, cfg *buildconfigs.BuildConfig) {
	(*hooks)[hookName] += getHookData(hookName, repo, cfg) + "\n"
}

func getHookData(hookName string, repo *buildconfigs.GitRepo, cfg *buildconfigs.BuildConfig) string {
	repoUrl, err := url.Parse(repo.URL)

	if err != nil {
		log.Errorf("Error parsing repo URL %s: %v", repo.URL, err)
		return ""
	}

	repoDirectory := strings.TrimSuffix(filepath.Base(filepath.Clean(repoUrl.Path)), ".git")

	repoPath := filepath.Join(getRepoStorageDir(cfg), repoDirectory)

	htmlFilename := filepath.Join(repoPath, hookName+".html")

	log.Infof("Getting hook data for %v, repo: %s filename: %v", hookName, repo.URL, htmlFilename)

	if _, statErr := os.Stat(htmlFilename); errors.Is(statErr, os.ErrNotExist) {
		log.Infof("No %s hook file found for repo %s", hookName, repo.URL)
		return ""
	}

	contents, err := os.ReadFile(htmlFilename)

	if err != nil {
		log.Errorf("Error reading %s hook file for repo %s: %v", hookName, repo.URL, err)
		return ""
	}

	if len(contents) == 0 {
		return ""
	}

	return string(contents)
}

func getRepoStorageDir(cfg *buildconfigs.BuildConfig) string {
	repoStorageDir := filepath.Join(filepath.Dir(cfg.Path), "repos")

	// Check if the directory exists
	if _, err := os.Stat(repoStorageDir); os.IsNotExist(err) {
		if err := os.MkdirAll(repoStorageDir, 0755); err != nil {
			log.Errorf("Error creating repo storage dir %s: %v", repoStorageDir, err)
		}
	}

	log.Infof("repoPath: %s", repoStorageDir)

	return repoStorageDir
}

func lnRepos(finalOutputDir string, repoStorageDir string) {
	repoLinkPath := filepath.Join(finalOutputDir, "repos")

	err := os.Symlink(repoStorageDir, repoLinkPath)

	if os.IsExist(err) {
		return
	} else if err != nil {
		log.Errorf("Error creating symlink for repos: %v", err)
		return
	}
}

func runVite(bctx *BuildContext, temporaryOutputDir string, finalOutputDir string) {
	if abortBuild(bctx.Ctx, bctx.BuildStatus, bctx.UpdateChannel) {
		return
	}

	req := &easyexec.ExecRequest{
		Executable:       "vite",
		Args:             []string{"build", "--outDir", finalOutputDir, "--base", "./"},
		WorkingDirectory: temporaryOutputDir,
		Log:              true,
		Timeout:          remainingSubprocessSeconds(bctx.Ctx),
	}

	res := easyexec.ExecWithRequest(req)

	if res.ExitCode != 0 {
		bctx.UpdateChannel <- fmt.Sprintf("Vite output:\n%s", res.Output)
		failBuild(bctx.BuildStatus, bctx.UpdateChannel, fmt.Sprintf("Vite build failed with exit code %d", res.ExitCode))
	} else {
		bctx.UpdateChannel <- "Vite build completed successfully"
	}
}

func getDirectorySizeHumanReadable(dir string) string {
	var size int64 = 0

	err := filepath.Walk(dir, func(_ string, info os.FileInfo, _ error) error {
		if info.IsDir() {
			return nil
		}
		size += info.Size()
		return nil
	})
	if err != nil {
		log.Errorf("Error calculating directory size: %v", err)
		return "unknown"
	}

	return formatSize(size)
}

func formatSize(size int64) string {
	switch {
	case size < 1024:
		return fmt.Sprintf("%d B", size)
	case size < 1024*1024:
		return fmt.Sprintf("%.2f KB", float64(size)/1024)
	case size < 1024*1024*1024:
		return fmt.Sprintf("%.2f MB", float64(size)/(1024*1024))
	default:
		return fmt.Sprintf("%.2f GB", float64(size)/(1024*1024*1024))
	}
}
func getBuildUrlBase() string {
	if os.Getenv("STENCILBOX_BUILD_URL_BASE") != "" {
		return os.Getenv("STENCILBOX_BUILD_URL_BASE")
	}

	return ""
}

func getWebuiUrlBase() string {
	return strings.TrimRight(os.Getenv("STENCILBOX_WEBUI_URL_BASE"), "/")
}

func webuiBuildConfigURL(base, configName string) string {
	path := "/webui/build-config/" + url.PathEscape(configName)
	if base == "" {
		return path
	}
	return strings.TrimRight(base, "/") + path
}

// failBuild marks the build as failed and streams the reason to the client.
func failBuild(buildStatus *BuildStatus, updateChannel chan string, message string) {
	buildStatus.IsError = true
	buildStatus.Message = message
	updateChannel <- message
}

// abortBuild reports context cancellation or deadline expiry and returns true if the build should stop.
func abortBuild(ctx context.Context, buildStatus *BuildStatus, updateChannel chan string) bool {
	if ctx.Err() == nil {
		return false
	}
	if buildStatus.IsError {
		return true
	}
	buildStatus.IsError = true
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		buildStatus.Message = fmt.Sprintf("Build exceeded the maximum time limit (%v)", totalBuildTimeout)
	} else {
		buildStatus.Message = "Build cancelled: " + ctx.Err().Error()
	}
	updateChannel <- buildStatus.Message
	return true
}

// remainingSubprocessSeconds returns seconds left before the build context deadline (minimum 10 for easyexec).
func remainingSubprocessSeconds(ctx context.Context) float64 {
	d, ok := ctx.Deadline()
	if !ok {
		return totalBuildTimeout.Seconds()
	}
	rem := time.Until(d)
	if rem <= 0 {
		return 10
	}
	s := rem.Seconds()
	if s < 10 {
		return 10
	}
	return s
}

func cloneRepos(bctx *BuildContext, outputDir string) {
	for _, repo := range bctx.BuildConfig.Repos {
		if abortBuild(bctx.Ctx, bctx.BuildStatus, bctx.UpdateChannel) {
			return
		}
		bctx.UpdateChannel <- fmt.Sprintf("Cloning repo %s", repo.URL)

		cloneRepo(bctx.Ctx, repo, outputDir)
	}
}

func cloneRepo(ctx context.Context, repo buildconfigs.GitRepo, outputDir string) {
	repoTimeout := math.Max(repo.Timeout, 60.0)
	if rem := remainingSubprocessSeconds(ctx); rem < repoTimeout {
		repoTimeout = rem
	}
	repo.Timeout = repoTimeout

	log.Infof("Cloning or pulling repo %s with timeout %f seconds", repo.URL, repo.Timeout)

	req := &git.CloneOrPullRequest{
		GitUrl:   repo.URL,
		LocalDir: outputDir,
		Timeout:  repo.Timeout,
		Log:      true,
	}

	res := git.CloneOrPull(req)

	if res.WasCloned {
		log.Infof("Cloned repo %s to %s", repo.URL, outputDir)
	} else {
		log.Infof("Pulled repo %s in %s", repo.URL, outputDir)
	}
}

func findLayersDir() string {
	dir, _ := dirs.GetFirstExistingDirectory("layers", []string{
		"../layers/",
		"/app/layers/",
		"/config/layers/",
	})

	if dir == "" {
		log.Warnf("No layers directory found, using default")
		return "/config/layers/"
	}

	return dir
}

func copyLayers(outputDir string) {
	layersDir := findLayersDir()

	layerBaseDir := filepath.Join(layersDir, "base")

	copyFile(layerBaseDir, outputDir, "style.css")
	copyFile(layerBaseDir, outputDir, "favicon.png")
}

func copyFile(fromDir string, toDir string, filename string) {
	contents, err := os.ReadFile(filepath.Join(fromDir, filename))

	if err != nil {
		log.Errorf("Error reading style.css: %v", err)
		return
	}

	if err := os.WriteFile(filepath.Join(toDir, filename), contents, 0644); err != nil {
		log.Errorf("Error writing %v: %v", filename, err)
		return
	}

	log.Infof("Copied %v to %v", filename, toDir)
}

// linkFaviconBaseURL returns the URL used to fetch a link's favicon. When url_internal is set it
// is used instead of url so favicons can be resolved from an internal endpoint that does not
// require reverse-proxy authentication.
func linkFaviconBaseURL(linkMap map[string]any) string {
	if internalURL, ok := linkMap["url_internal"].(string); ok && internalURL != "" {
		return internalURL
	}
	if linkURL, ok := linkMap["url"].(string); ok {
		return linkURL
	}
	return ""
}

// countFaviconFetchJobs returns how many links will run a remote favicon fetch (same preconditions as the fetch loop).
func countFaviconFetchJobs(dataMap map[string]any, iconsDir string) int {
	n := 0
	for _, linkMap := range categoryLinkMaps(dataMap) {
		if needsFaviconFetch(linkMap, iconsDir) {
			n++
		}
	}
	return n
}

// needsFaviconFetch reports whether a link will run a remote favicon fetch.
func needsFaviconFetch(linkMap map[string]any, iconsDir string) bool {
	linkURL, ok := linkMap["url"].(string)
	if !ok || linkURL == "" || hasExplicitIcon(linkMap) {
		return false
	}
	return findExistingIconFile(iconsDir, sanitizeFilename(linkURL)) == ""
}

// mapsIn returns the map items of a YAML list, skipping any other items.
func mapsIn(v any) []map[string]any {
	items, _ := v.([]any)
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

// categoryLinkMaps returns every link of every category in links data.
func categoryLinkMaps(dataMap map[string]any) []map[string]any {
	var out []map[string]any
	for _, category := range mapsIn(dataMap["categories"]) {
		out = append(out, mapsIn(category["links"])...)
	}
	return out
}
func processLinksWithFavicons(ctx context.Context, linksData any, outputDir string, updateChannel chan string) (any, error) {
	// Create icons directory
	iconsDir := filepath.Join(outputDir, "icons")
	if err := os.MkdirAll(iconsDir, 0o755); err != nil {
		return linksData, fmt.Errorf("failed to create icons directory: %w", err)
	}

	// Copy placeholder icon once
	placeholderIconPath := ensurePlaceholderIconOrWarn(iconsDir)

	// Convert to map for processing
	dataMap, ok := linksData.(map[string]any)
	if !ok {
		return linksData, errors.New("links data is not a map")
	}

	normalizeLinksDataURLs(dataMap)

	totalFaviconJobs := countFaviconFetchJobs(dataMap, iconsDir)
	faviconJob := 0

	for _, linkMap := range categoryLinkMaps(dataMap) {
		if err := ctx.Err(); err != nil {
			return linksData, err
		}

		applyFaviconToLink(linkMap, iconsDir, placeholderIconPath, updateChannel, &faviconJob, totalFaviconJobs)
	}

	return dataMap, nil
}

func ensurePlaceholderIconOrWarn(iconsDir string) string {
	placeholderIconPath := ensurePlaceholderIcon(iconsDir)
	if placeholderIconPath == "" {
		log.Warnf("Failed to copy placeholder icon, links without favicons will not have icons")
	}
	return placeholderIconPath
}
func hasExplicitIcon(linkMap map[string]any) bool {
	existingIcon, hasIcon := linkMap["icon"]
	if !hasIcon {
		return false
	}
	iconStr, ok := existingIcon.(string)
	return ok && iconStr != ""
}

func applyFaviconToLink(linkMap map[string]any, iconsDir, placeholder string, updateChannel chan string, job *int, total int) {
	linkURL, ok := linkMap["url"].(string)
	if !ok || linkURL == "" || hasExplicitIcon(linkMap) {
		return
	}

	safeFilename := sanitizeFilename(linkURL)
	if assignCachedFavicon(linkMap, iconsDir, safeFilename, linkURL, updateChannel) {
		return
	}

	fetchAndAssignFavicon(linkMap, iconsDir, placeholder, safeFilename, updateChannel, job, total)
}

func assignCachedFavicon(linkMap map[string]any, iconsDir, safeFilename, linkURL string, updateChannel chan string) bool {
	existing := findExistingIconFile(iconsDir, safeFilename)
	if existing == "" {
		return false
	}
	iconPath := "icons/" + existing
	linkMap["icon"] = iconPath
	updateChannel <- fmt.Sprintf("Using cached favicon for %s: %s", linkURL, iconPath)
	return true
}

func fetchAndAssignFavicon(linkMap map[string]any, iconsDir, placeholder, safeFilename string, updateChannel chan string, job *int, total int) {
	*job++
	base := linkFaviconBaseURL(linkMap)
	updateChannel <- fmt.Sprintf("Fetching favicon (%d of %d): %s", *job, total, base)
	got, err := scraper.FindAndDownloadFavicon(base, iconsDir, safeFilename)
	if err != nil {
		log.Debugf("Failed to get favicon for %s: %v", base, err)
		updateChannel <- fmt.Sprintf("No favicon found for %s: %v", base, err)
		if placeholder != "" {
			linkMap["icon"] = placeholder
		}
		return
	}
	updateChannel <- faviconFoundMessage(got)
	linkMap["icon"] = "icons/" + got.Filename
}

func faviconFoundMessage(got scraper.DownloadedFavicon) string {
	if got.MimeType == "" {
		return "Found favicon: " + got.SourceURL
	}
	return fmt.Sprintf("Found favicon: %s (%s)", got.SourceURL, got.MimeType)
}

// normalizeLinksDataURLs ensures scheme-less link URLs become absolute https:// hrefs.
func normalizeLinksDataURLs(dataMap map[string]any) {
	for _, linkMap := range categoryLinkMaps(dataMap) {
		normalizeLinkURL(linkMap)
	}

	for _, linkMap := range mapsIn(dataMap["links"]) {
		normalizeLinkURL(linkMap)
	}
}

func normalizeLinkURL(linkMap map[string]any) {
	linkURL, ok := linkMap["url"].(string)
	if !ok || linkURL == "" {
		return
	}
	linkMap["url"] = scraper.NormalizeURL(linkURL)
}
func ensurePlaceholderIcon(iconsDir string) string {
	placeholderFilename := "placeholder.png"
	placeholderPath := filepath.Join(iconsDir, placeholderFilename)

	// Check if placeholder already exists
	if _, err := os.Stat(placeholderPath); err == nil {
		return "icons/" + placeholderFilename
	}

	// Try to copy favicon.png from layers/base as placeholder
	layersDir := findLayersDir()
	layerBaseDir := filepath.Join(layersDir, "base")
	sourcePath := filepath.Join(layerBaseDir, "favicon.png")

	// Read the source favicon
	contents, err := os.ReadFile(sourcePath)
	if err != nil {
		log.Debugf("Failed to read placeholder icon from %s: %v", sourcePath, err)
		return ""
	}

	// Write to icons directory
	err = os.WriteFile(placeholderPath, contents, 0644)
	if err != nil {
		log.Debugf("Failed to write placeholder icon: %v", err)
		return ""
	}

	log.Debugf("Copied placeholder icon to %s", placeholderPath)
	return "icons/" + placeholderFilename
}

func sanitizeFilename(urlStr string) string {
	// Remove protocol
	urlStr = strings.TrimPrefix(urlStr, "http://")
	urlStr = strings.TrimPrefix(urlStr, "https://")

	// Replace invalid filename characters
	urlStr = strings.ReplaceAll(urlStr, "/", "_")
	urlStr = strings.ReplaceAll(urlStr, ":", "_")
	urlStr = strings.ReplaceAll(urlStr, "?", "_")
	urlStr = strings.ReplaceAll(urlStr, "&", "_")
	urlStr = strings.ReplaceAll(urlStr, "=", "_")
	urlStr = strings.ReplaceAll(urlStr, "#", "_")
	urlStr = strings.ReplaceAll(urlStr, "%", "_")

	// Limit length
	if len(urlStr) > 100 {
		urlStr = urlStr[:100]
	}

	return urlStr
}

func findExistingIconFile(iconsDir, baseName string) string {
	if baseName == "" {
		return ""
	}

	if _, err := os.Stat(filepath.Join(iconsDir, baseName)); err == nil {
		return baseName
	}

	for _, ext := range []string{".svg", ".png", ".jpg", ".jpeg", ".ico", ".gif", ".webp"} {
		filename := baseName + ext
		if _, err := os.Stat(filepath.Join(iconsDir, filename)); err == nil {
			return filename
		}
	}

	return ""
}

func readDatafile(path string, buildconfigPath string) (any, error) {
	path = filepath.Join(buildconfigPath, path)

	content, err := os.ReadFile(path)

	if err != nil {
		return nil, fmt.Errorf("failed to read datafile: %v", path)
	}

	var data any

	err = yaml.Unmarshal(content, &data)

	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal datafile: %v", path)
	}

	return data, nil
}
