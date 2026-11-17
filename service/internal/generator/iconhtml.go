package generator

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/jamesread/StencilBox/internal/scraper"
)

func getNewTemplater(outputDir string) *template.Template {
	funcMap := template.FuncMap{
		"upper":          strings.ToUpper,
		"lower":          strings.ToLower,
		"replace":        strings.ReplaceAll,
		"formatDateTime": formatDateTime,
		"safeURL":        safeURL,
		"list":           list,
		"dict":           dict,
		"where":          where,
		"whereNot":       whereNot,
		"whereIn":        whereIn,
		"pluck":          pluck,
		"overdue":        overdue,
		"dueLabel":       dueLabel,
		"isOverdue":      isOverdue,
		"linkIconHTML": func(iconPath, cssClass, extraAttrs string) string {
			return renderLinkIconHTML(outputDir, iconPath, cssClass, extraAttrs)
		},
	}

	return template.New("index.html").Funcs(funcMap).Option("missingkey=zero")
}

func renderLinkIconHTML(outputDir, iconPath, cssClass, extraAttrs string) string {
	if iconPath == "" {
		return ""
	}

	escapedPath := html.EscapeString(iconPath)
	classAttr := iconClassAttr(cssClass)

	if isRemoteIcon(iconPath) {
		return remoteIconHTML(iconPath, escapedPath, classAttr, extraAttrs)
	}

	localPath := filepath.Join(outputDir, iconPath)
	if isSVGIconPath(iconPath) || scraper.IsSVGContent(localPath) {
		return localSVGIconHTML(localPath, escapedPath, cssClass, classAttr, extraAttrs)
	}

	return imgIconHTML(escapedPath, classAttr, extraAttrs)
}

func iconClassAttr(cssClass string) string {
	if cssClass == "" {
		return ""
	}
	return fmt.Sprintf(` class="%s"`, html.EscapeString(cssClass))
}

func objectIconHTML(escapedPath, classAttr, extraAttrs string) string {
	return fmt.Sprintf(
		`<object data="%s" type="image/svg+xml"%s %s></object>`,
		escapedPath, classAttr, extraAttrs,
	)
}

func imgIconHTML(escapedPath, classAttr, extraAttrs string) string {
	return fmt.Sprintf(
		`<img src="%s" alt=""%s %s />`,
		escapedPath, classAttr, extraAttrs,
	)
}

func remoteIconHTML(iconPath, escapedPath, classAttr, extraAttrs string) string {
	if isSVGIconPath(iconPath) {
		return objectIconHTML(escapedPath, classAttr, extraAttrs)
	}
	return imgIconHTML(escapedPath, classAttr, extraAttrs)
}

// localSVGIconHTML inlines a local SVG file, or falls back to an object tag when it cannot be read.
func localSVGIconHTML(localPath, escapedPath, cssClass, classAttr, extraAttrs string) string {
	data, err := os.ReadFile(localPath)
	if err != nil {
		return objectIconHTML(escapedPath, classAttr, extraAttrs)
	}

	wrapperClass := cssClass
	if wrapperClass == "" {
		wrapperClass = "link-icon"
	}
	return fmt.Sprintf(
		`<span class="%s link-icon-svg" %s>%s</span>`,
		html.EscapeString(wrapperClass), extraAttrs, string(data),
	)
}
func isRemoteIcon(iconPath string) bool {
	return strings.HasPrefix(iconPath, "http://") || strings.HasPrefix(iconPath, "https://")
}

func isSVGIconPath(iconPath string) bool {
	path := strings.ToLower(iconPath)
	if idx := strings.Index(path, "?"); idx != -1 {
		path = path[:idx]
	}
	return strings.HasSuffix(path, ".svg")
}
