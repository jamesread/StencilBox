package scraper

import (
	"encoding/base64"
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

func normalizeURL(rawURL string) string {
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
		Timeout: 10 * time.Second,
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
	if faviconURL == "" {
		return ""
	}

	// If favicon URL is absolute, return as-is
	if strings.HasPrefix(faviconURL, "http://") || strings.HasPrefix(faviconURL, "https://") {
		return faviconURL
	}

	// Parse the page URL to resolve relative favicon URLs
	parsedPageURL, err := url.Parse(pageURL)
	if err != nil {
		log.Warnf("Failed to parse page URL %s: %v", pageURL, err)
		return faviconURL
	}

	// Resolve relative URL
	resolvedURL := parsedPageURL.ResolveReference(&url.URL{Path: faviconURL})
	return resolvedURL.String()
}

// GetFaviconURL fetches the favicon URL from a webpage
func GetFaviconURL(pageURL string) (string, error) {
	normalizedURL := normalizeURL(pageURL)
	if normalizedURL == "" {
		return "", fmt.Errorf("invalid URL: %s", pageURL)
	}

	content, err := fetchPageContent(normalizedURL)
	if err != nil {
		return "", err
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	faviconPath := searchLinksForFavicon(doc.Find("link"))
	if faviconPath == "" {
		// Try default favicon location
		parsedURL, err := url.Parse(normalizedURL)
		if err == nil {
			faviconPath = parsedURL.Scheme + "://" + parsedURL.Host + "/favicon.ico"
		}
	}

	if faviconPath == "" {
		return "", fmt.Errorf("no favicon found")
	}

	// Resolve relative favicon URLs
	faviconURL := resolveFaviconURL(faviconPath, normalizedURL)
	return faviconURL, nil
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
func DownloadFavicon(faviconURL, saveDir, filename string) (string, error) {
	// Create icons directory if it doesn't exist
	err := os.MkdirAll(saveDir, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create icons directory: %w", err)
	}

	var faviconData []byte
	var mimeType string

	// Check if this is a base64 data URL
	if strings.HasPrefix(faviconURL, "data:") {
		mimeType, faviconData, err = decodeDataURL(faviconURL)
		if err != nil {
			return "", fmt.Errorf("failed to decode data URL: %w", err)
		}
	} else {
		// Regular HTTP URL - download it
		client := &http.Client{
			Timeout: 10 * time.Second,
		}
		req, err := http.NewRequest("GET", faviconURL, nil)
		if err != nil {
			return "", fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; StencilBox/1.0)")

		resp, err := client.Do(req)
		if err != nil {
			return "", fmt.Errorf("failed to download favicon: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		// Read favicon data
		faviconData, err = io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to read favicon data: %w", err)
		}

		// Get MIME type from response header
		mimeType = resp.Header.Get("Content-Type")
	}

	// Determine file extension from filename, MIME type, or default
	ext := filepath.Ext(filename)
	if ext == "" {
		if mimeType != "" {
			// Try to get extension from MIME type
			exts, err := mime.ExtensionsByType(mimeType)
			if err == nil && len(exts) > 0 {
				ext = exts[0] // Use first extension
			}
		}

		// Fallback to manual detection if mime package didn't help
		if ext == "" {
			if strings.Contains(mimeType, "png") {
				ext = ".png"
			} else if strings.Contains(mimeType, "svg") {
				ext = ".svg"
			} else if strings.Contains(mimeType, "jpeg") || strings.Contains(mimeType, "jpg") {
				ext = ".jpg"
			} else if strings.Contains(mimeType, "ico") {
				ext = ".ico"
			} else {
				ext = ".ico" // default
			}
		}
		filename = filename + ext
	}

	savePath := filepath.Join(saveDir, filename)

	// Save to file
	err = os.WriteFile(savePath, faviconData, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to save favicon: %w", err)
	}

	return filename, nil
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

func searchLinksForFavicon(sel *goquery.Selection) string {
	for _, node := range sel.Nodes {
		favicon := getFavicon(node)

		if favicon != "" {
			return favicon
		}
	}

	return ""
}
