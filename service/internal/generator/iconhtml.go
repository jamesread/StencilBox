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
		"upper":       strings.ToUpper,
		"lower":       strings.ToLower,
		"replace":     strings.ReplaceAll,
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
	escapedClass := html.EscapeString(cssClass)
	classAttr := ""
	if cssClass != "" {
		classAttr = fmt.Sprintf(` class="%s"`, escapedClass)
	}

	if isRemoteIcon(iconPath) {
		if isSVGIconPath(iconPath) {
			return fmt.Sprintf(
				`<object data="%s" type="image/svg+xml"%s %s></object>`,
				escapedPath, classAttr, extraAttrs,
			)
		}
		return fmt.Sprintf(
			`<img src="%s" alt=""%s %s />`,
			escapedPath, classAttr, extraAttrs,
		)
	}

	localPath := filepath.Join(outputDir, iconPath)
	if isSVGIconPath(iconPath) || scraper.IsSVGContent(localPath) {
		data, err := os.ReadFile(localPath)
		if err != nil {
			return fmt.Sprintf(
				`<object data="%s" type="image/svg+xml"%s %s></object>`,
				escapedPath, classAttr, extraAttrs,
			)
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

	return fmt.Sprintf(
		`<img src="%s" alt=""%s %s />`,
		escapedPath, classAttr, extraAttrs,
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
