package main

import (
	"github.com/jamesread/StencilBox/internal/config"
//	"github.com/jamesread/StencilBox/internal/generator"
	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
	"io"
	"fmt"
	"net/http"
	log "github.com/sirupsen/logrus"
	"strings"
)

func curl(url string) string {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil);

	if err != nil {
		log.Fatal(err)
		return ""
	}

	resp, err := client.Do(req)

	if err != nil {
		log.Fatal(err)
		return ""
	}

	defer resp.Body.Close()

	bodyText, err := io.ReadAll(resp.Body)

	if err != nil {
		log.Fatal(err)
		return ""
	}

	return fmt.Sprintf("%s", bodyText)
}

func getFavicon(node *html.Node) string {
	isIcon := false
	href := ""

	for _, attr := range node.Attr {

		if attr.Key == "rel" && attr.Val == "icon" {
			isIcon = true
		}

		if attr.Key == "href" {
			href = attr.Val
		}

		if isIcon && href != "" {
			return href
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

func processUrl(site *config.Site) {
	content := curl(site.URL)

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))

	if err != nil {
		log.Errorf("%v", err)
		return
	}

	favicon := searchLinksForFavicon(doc.Find("link"))

	log.Infof("%v (%v) = %v", site.Name, site.URL, favicon)
}

func main() {
	cfg := config.ReadConfigFile()

	for _, site := range cfg.Sites {
		processUrl(site)
	}

	//generator.Generate(config.ReadConfigFile())
}
