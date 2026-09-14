package scraper

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
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
	client := &http.Client{
		Timeout: faviconFetchTimeout,
	}
	req, err := http.NewRequest("GET", pageURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; StencilBox/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return string(bodyText), nil
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

// decodeDataURL decodes a base64 data URL and returns the MIME type and decoded data
func decodeDataURL(dataURL string) (mimeType string, data []byte, err error) {
	// Data URLs have the format: data:[<mediatype>][;base64],<data>
	if !strings.HasPrefix(dataURL, "data:") {
		return "", nil, fmt.Errorf("not a data URL")
	}

	// Remove "data:" prefix
	dataURL = dataURL[5:]

	// Find the comma that separates metadata from data
	commaIdx := strings.Index(dataURL, ",")
	if commaIdx == -1 {
		return "", nil, fmt.Errorf("invalid data URL format: missing comma")
	}

	// Extract metadata and data parts
	metadata := dataURL[:commaIdx]
	dataPart := dataURL[commaIdx+1:]

	// Parse MIME type (may include parameters like charset)
	mimeType = "image/png" // default
	if strings.HasPrefix(metadata, "image/") {
		// Extract MIME type (before any semicolon)
		mimeParts := strings.Split(metadata, ";")
		if len(mimeParts) > 0 {
			mimeType = strings.TrimSpace(mimeParts[0])
		}
	}

	// Check if it's base64 encoded
	if strings.Contains(metadata, "base64") {
		// Decode base64 data
		decoded, err := base64.StdEncoding.DecodeString(dataPart)
		if err != nil {
			return "", nil, fmt.Errorf("failed to decode base64 data: %w", err)
		}
		return mimeType, decoded, nil
	}

	// URL-encoded data (less common for images, but handle it)
	decoded, err := url.QueryUnescape(dataPart)
	if err != nil {
		return "", nil, fmt.Errorf("failed to decode URL-encoded data: %w", err)
	}
	return mimeType, []byte(decoded), nil
}

// DownloadFavicon downloads a favicon and saves it to the specified directory
// It supports both regular HTTP URLs and base64 data URLs
func DownloadFavicon(faviconURL, saveDir, filename string) (DownloadedFavicon, error) {
	empty := DownloadedFavicon{}
	err := os.MkdirAll(saveDir, 0755)
	if err != nil {
		return empty, fmt.Errorf("failed to create icons directory: %w", err)
	}

	var faviconData []byte
	var mimeType string

	if strings.HasPrefix(faviconURL, "data:") {
		mimeType, faviconData, err = decodeDataURL(faviconURL)
		if err != nil {
			return empty, fmt.Errorf("failed to decode data URL: %w", err)
		}
	} else {
		client := &http.Client{
			Timeout: faviconFetchTimeout,
		}
		req, err := http.NewRequest("GET", faviconURL, nil)
		if err != nil {
			return empty, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; StencilBox/1.0)")

		resp, err := client.Do(req)
		if err != nil {
			return empty, fmt.Errorf("failed to download favicon: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return empty, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		faviconData, err = io.ReadAll(resp.Body)
		if err != nil {
			return empty, fmt.Errorf("failed to read favicon data: %w", err)
		}

		mimeType = resp.Header.Get("Content-Type")
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

	err = os.WriteFile(savePath, faviconData, 0644)
	if err != nil {
		return empty, fmt.Errorf("failed to save favicon: %w", err)
	}

	return DownloadedFavicon{Filename: filename, SourceURL: faviconURL, MimeType: mimeType}, nil
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
	if mimeType != "" {
		mediaType, _, _ := strings.Cut(mimeType, ";")
		mediaType = strings.TrimSpace(mediaType)
		exts, err := mime.ExtensionsByType(mediaType)
		if err == nil && len(exts) > 0 {
			return exts[0]
		}
	}
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
	var relValues []string
	href := ""

	for _, attr := range node.Attr {
		if attr.Key == "rel" {
			relValues = strings.Fields(attr.Val)
		}
		if attr.Key == "href" {
			href = attr.Val
		}
	}

	// Check if this link has icon-related rel values
	for _, rel := range relValues {
		if rel == "icon" || rel == "shortcut" || rel == "apple-touch-icon" {
			if href != "" {
				return href
			}
		}
	}

	return ""
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
