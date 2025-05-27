package scraper

import (
	"net/http"
	log "github.com/sirupsen/logrus"
	"strings"
	"github.com/PuerkitoBio/goquery"
	"io"
	"fmt"
	"golang.org/x/net/html"
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

func ProcessUrl(url string) {
	content := curl(url)

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))

	if err != nil {
		log.Errorf("%v", err)
		return
	}

	favicon := searchLinksForFavicon(doc.Find("link"))

	log.Infof("%v = %v", url, favicon)
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
