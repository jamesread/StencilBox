package scraper

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/html"
)

// ErrRejectedMime is returned when a downloaded favicon is not an image or SVG.
var ErrRejectedMime = errors.New("favicon is not an image or svg")

// faviconFetchTimeout bounds each HTTP request used when resolving and downloading link favicons during builds.
const faviconFetchTimeout = 8 * time.Second

func NormalizeURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}

	// Add https:// if no scheme is present
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}

	return rawURL
}

func fetchPageContent(pageURL string) (string, error) {
	body, _, err := fetchBody(pageURL, "fetch page", "read response")
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// fetchBody GETs a URL, bounded by faviconFetchTimeout, and returns the body and its
// Content-Type. fetchLabel and readLabel say what was being done when a step fails.
func fetchBody(rawURL, fetchLabel, readLabel string) ([]byte, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), faviconFetchTimeout)
	defer cancel()

	client := &http.Client{
		Timeout: faviconFetchTimeout,
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; StencilBox/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to %s: %w", fetchLabel, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to %s: %w", readLabel, err)
	}

	return body, resp.Header.Get("Content-Type"), nil
}
func resolveFaviconURL(faviconURL, pageURL string) string {
	if faviconURL == "" || strings.HasPrefix(faviconURL, "data:") {
		return faviconURL
	}

	base, err := url.Parse(pageURL)
	if err != nil {
		log.Warnf("Failed to parse page URL %s: %v", pageURL, err)
		return faviconURL
	}

	ref, err := url.Parse(faviconURL)
	if err != nil {
		return faviconURL
	}

	return base.ResolveReference(ref).String()
}

func isFaviconMimeType(mimeType string) bool {
	mediaType, _, _ := strings.Cut(mimeType, ";")
	mediaType = strings.TrimSpace(strings.ToLower(mediaType))
	if mediaType == "" {
		return false
	}
	if strings.HasPrefix(mediaType, "image/") {
		return true
	}
	return strings.Contains(mediaType, "svg")
}

func defaultFaviconURL(pageURL string) string {
	parsedURL, err := url.Parse(pageURL)
	if err != nil || parsedURL.Host == "" {
		return ""
	}
	return parsedURL.Scheme + "://" + parsedURL.Host + "/favicon.ico"
}

func addFaviconCandidate(out []string, seen map[string]bool, raw, pageURL string) []string {
	abs := resolveFaviconURL(raw, pageURL)
	if abs == "" || seen[abs] {
		return out
	}
	seen[abs] = true
	return append(out, abs)
}

func faviconCandidates(pageURL string) ([]string, error) {
	normalizedURL := NormalizeURL(pageURL)
	if normalizedURL == "" {
		return nil, fmt.Errorf("invalid URL: %s", pageURL)
	}

	seen := make(map[string]bool)
	candidates := appendLinkFavicons(nil, seen, normalizedURL)
	candidates = addFaviconCandidate(candidates, seen, defaultFaviconURL(normalizedURL), normalizedURL)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no favicon found")
	}
	return candidates, nil
}

func appendLinkFavicons(candidates []string, seen map[string]bool, pageURL string) []string {
	content, err := fetchPageContent(pageURL)
	if err != nil {
		return candidates
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		return candidates
	}
	for _, node := range doc.Find("link").Nodes {
		candidates = addFaviconCandidate(candidates, seen, getFavicon(node), pageURL)
	}
	return candidates
}

// GetFaviconURL fetches the first favicon URL from a webpage.
func GetFaviconURL(pageURL string) (string, error) {
	candidates, err := faviconCandidates(pageURL)
	if err != nil {
		return "", err
	}
	return candidates[0], nil
}

// DownloadedFavicon is a favicon saved from a remote icon URL.
type DownloadedFavicon struct {
	Filename  string
	SourceURL string
	MimeType  string
}

// FindAndDownloadFavicon tries each discovered favicon URL until one is an image or SVG.
func FindAndDownloadFavicon(pageURL, saveDir, filename string) (DownloadedFavicon, error) {
	candidates, err := faviconCandidates(pageURL)
	if err != nil {
		return DownloadedFavicon{}, err
	}

	return downloadFirstImageFavicon(candidates, saveDir, filename)
}

func downloadFirstImageFavicon(candidates []string, saveDir, filename string) (DownloadedFavicon, error) {
	var lastErr error
	for _, candidate := range candidates {
		log.WithField("path", candidate).Info("Found favicon")
		saved, err := DownloadFavicon(candidate, saveDir, filename)
		if err == nil {
			return saved, nil
		}
		lastErr = fmt.Errorf("%s: %w", candidate, err)
	}
	if lastErr != nil {
		return DownloadedFavicon{}, lastErr
	}
	return DownloadedFavicon{}, fmt.Errorf("no favicon found")
}

// decodeDataURL decodes a data URL and returns the MIME type and decoded data
func decodeDataURL(dataURL string) (mimeType string, data []byte, err error) {
	// Data URLs have the format: data:[<mediatype>][;base64],<data>
	rest, ok := strings.CutPrefix(dataURL, "data:")
	if !ok {
		return "", nil, errors.New("not a data URL")
	}

	// The comma separates metadata from data
	metadata, dataPart, found := strings.Cut(rest, ",")
	if !found {
		return "", nil, errors.New("invalid data URL format: missing comma")
	}

	data, err = decodeDataPayload(metadata, dataPart)
	if err != nil {
		return "", nil, err
	}

	return dataURLMimeType(metadata), data, nil
}

// dataURLMimeType returns the MIME type in a data URL's metadata (it may include parameters
// like charset), defaulting to image/png.
func dataURLMimeType(metadata string) string {
	if strings.HasPrefix(metadata, "image/") {
		// Extract MIME type (before any semicolon)
		mimeType, _, _ := strings.Cut(metadata, ";")
		return strings.TrimSpace(mimeType)
	}

	return "image/png"
}

func decodeDataPayload(metadata, payload string) ([]byte, error) {
	if strings.Contains(metadata, "base64") {
		decoded, err := base64.StdEncoding.DecodeString(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to decode base64 data: %w", err)
		}
		return decoded, nil
	}

	// URL-encoded data (less common for images, but handle it)
	decoded, err := url.QueryUnescape(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to decode URL-encoded data: %w", err)
	}
	return []byte(decoded), nil
}

// DownloadFavicon downloads a favicon and saves it to the specified directory
// It supports both regular HTTP URLs and base64 data URLs
func DownloadFavicon(faviconURL, saveDir, filename string) (DownloadedFavicon, error) {
	empty := DownloadedFavicon{}
	if err := os.MkdirAll(saveDir, 0o755); err != nil {
		return empty, fmt.Errorf("failed to create icons directory: %w", err)
	}

	mimeType, faviconData, err := loadFaviconData(faviconURL)
	if err != nil {
		return empty, err
	}

	log.WithFields(log.Fields{
		"path":     faviconURL,
		"mimeType": mimeType,
	}).Info("Downloaded favicon")

	if !isFaviconMimeType(mimeType) {
		return empty, fmt.Errorf("%w: %s", ErrRejectedMime, mimeType)
	}

	filename = filenameWithImageExt(filename, faviconURL, mimeType)
	savePath := filepath.Join(saveDir, filename)

	if err = os.WriteFile(savePath, faviconData, 0o644); err != nil {
		return empty, fmt.Errorf("failed to save favicon: %w", err)
	}

	return DownloadedFavicon{Filename: filename, SourceURL: faviconURL, MimeType: mimeType}, nil
}

// loadFaviconData returns the favicon's MIME type and bytes, from a data URL or over HTTP.
func loadFaviconData(faviconURL string) (string, []byte, error) {
	if strings.HasPrefix(faviconURL, "data:") {
		mimeType, data, err := decodeDataURL(faviconURL)
		if err != nil {
			return "", nil, fmt.Errorf("failed to decode data URL: %w", err)
		}
		return mimeType, data, nil
	}

	data, mimeType, err := fetchBody(faviconURL, "download favicon", "read favicon data")
	return mimeType, data, err
}

func filenameWithImageExt(filename, faviconURL, mimeType string) string {
	if imageFileExt(filename) != "" {
		return filename
	}
	ext := imageFileExt(faviconURL)
	if ext == "" {
		ext = mimeTypeFileExt(mimeType)
	}
	return filename + ext
}

func mimeTypeFileExt(mimeType string) string {
	if ext := registeredMimeExt(mimeType); ext != "" {
		return ext
	}
	return guessedMimeExt(mimeType)
}

func registeredMimeExt(mimeType string) string {
	if mimeType == "" {
		return ""
	}
	mediaType, _, _ := strings.Cut(mimeType, ";")
	exts, err := mime.ExtensionsByType(strings.TrimSpace(mediaType))
	if err == nil && len(exts) > 0 {
		return exts[0]
	}
	return ""
}

func guessedMimeExt(mimeType string) string {
	switch {
	case strings.Contains(mimeType, "png"):
		return ".png"
	case strings.Contains(mimeType, "svg"):
		return ".svg"
	case strings.Contains(mimeType, "jpeg"), strings.Contains(mimeType, "jpg"):
		return ".jpg"
	default:
		return ".ico"
	}
}

// ProcessUrl is kept for backward compatibility but now uses GetFaviconURL
func ProcessUrl(url string) {
	favicon, err := GetFaviconURL(url)
	if err != nil {
		log.Errorf("Failed to get favicon for %s: %v", url, err)
		return
	}
	log.Infof("%v = %v", url, favicon)
}

func getFavicon(node *html.Node) string {
	relValues, href := linkAttributes(node)

	// Check if this link has icon-related rel values
	if href != "" && slices.ContainsFunc(relValues, isIconRel) {
		return href
	}

	return ""
}

// linkAttributes returns the rel values and the href of a link element.
func linkAttributes(node *html.Node) ([]string, string) {
	var relValues []string
	href := ""

	for _, attr := range node.Attr {
		switch attr.Key {
		case "rel":
			relValues = strings.Fields(attr.Val)
		case "href":
			href = attr.Val
		}
	}

	return relValues, href
}

func isIconRel(rel string) bool {
	return rel == "icon" || rel == "shortcut" || rel == "apple-touch-icon"
}

var knownImageExtensions = map[string]bool{
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".gif":  true,
	".svg":  true,
	".ico":  true,
	".webp": true,
}

func imageFileExt(path string) string {
	path = strings.Split(path, "?")[0]
	path = strings.Split(path, "#")[0]
	ext := strings.ToLower(filepath.Ext(path))
	if knownImageExtensions[ext] {
		return ext
	}
	return ""
}

// IsSVGContent reports whether path points to a file whose contents look like SVG.
func IsSVGContent(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	trimmed := strings.TrimSpace(string(data))
	if strings.HasPrefix(trimmed, "<svg") {
		return true
	}
	if strings.HasPrefix(trimmed, "<?xml") {
		return strings.Contains(strings.ToLower(trimmed), "<svg")
	}
	return false
}
