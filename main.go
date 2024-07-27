package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/chromedp/chromedp"
)

type TargetUrl struct {
	Url                   string
	SelectorToWaitVisible string
}

func NewTargetUrl(url, selectorToWaitVisible string) *TargetUrl {
	return &TargetUrl{
		Url:                   url,
		SelectorToWaitVisible: selectorToWaitVisible,
	}
}

func isRokuDocsLinkValid(link *string) bool {
	if *link == "" {
		return false
	}
	if !strings.HasPrefix(*link, "https://developer.roku.com/") {
		return false
	}
	if !strings.Contains(*link, "/docs/") {
		return false
	}
	return true
}

func sanitizeRokuDocsLink(link string) string {
	splitLink := strings.Split(link, "/")
	if splitLink[3] != "docs" {
		link = strings.Join(append(splitLink[:3], splitLink[4:]...), "/")
	}

	anchorIndex := strings.Index(link, "#")

	if anchorIndex != -1 {
		link = link[:anchorIndex]
	}

	return link
}

func main() {
	allocatorContext, cancel := chromedp.NewExecAllocator(context.Background(), append(
		chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
	)...)
	defer cancel()

	targetUrls := []TargetUrl{
		*NewTargetUrl(
			"https://developer.roku.com/docs/features/features-overview.md",
			"#document-nav-menu > nav:nth-child(1) > div:nth-child(2) > div:nth-child(1) > ul:nth-child(1) > li:nth-child(1) > a:nth-child(1)",
		),
		*NewTargetUrl(
			"https://developer.roku.com/docs/specs/specs-overview.md",
			"#document-nav-menu > nav:nth-child(1) > div:nth-child(2) > div:nth-child(1) > ul:nth-child(1) > li:nth-child(1) > a:nth-child(1)",
		),
		*NewTargetUrl(
			"https://developer.roku.com/docs/developer-program/getting-started/roku-dev-prog.md",
			"#document-nav-menu > nav:nth-child(1) > div:nth-child(2) > div:nth-child(1) > ul:nth-child(1) > li:nth-child(1) > div:nth-child(2) > ul:nth-child(1) > li:nth-child(1)",
		),
		*NewTargetUrl(
			"https://developer.roku.com/docs/references/references-overview.md",
			"#document-nav-menu > nav:nth-child(1) > div:nth-child(2) > div:nth-child(1) > ul:nth-child(1) > li:nth-child(1) > a:nth-child(1)",
		),
	}

	var wg sync.WaitGroup
	var mux sync.Mutex
	var pageLinks []string

	for _, targetUrl := range targetUrls {
		wg.Add(1)
		go func(targetUrl TargetUrl) {
			defer wg.Done()

			fmt.Printf("Started: %s\n", targetUrl.Url)

			ctx, cancel := chromedp.NewContext(allocatorContext)
			defer cancel()

			var links []string
			err := chromedp.Run(ctx,
				chromedp.Navigate(targetUrl.Url),
				chromedp.WaitVisible(targetUrl.SelectorToWaitVisible),
				chromedp.Evaluate(`Array.from(document.querySelectorAll('a')).map(a => a.href)`, &links),
			)
			if err != nil {
				fmt.Println("Failed to scrape:", targetUrl.Url, err)
				return
			}

			mux.Lock()
			// TODO: Is the TargetUrl.Url also included in pageLinks?
			pageLinks = append(pageLinks, links...)
			mux.Unlock()

			fmt.Printf("Finished: %s\n", targetUrl.Url)
		}(targetUrl)
	}

	wg.Wait()

	linksSet := make(map[string]struct{})
	var exists = struct{}{}

	for _, link := range pageLinks {
		if isRokuDocsLinkValid(&link) {
			linksSet[sanitizeRokuDocsLink(link)] = exists
		}
	}

	outputFile, err := os.Create("output.txt")

	if err != nil {
		log.Fatalf("Failed to create file: %v", err)
	}
	defer outputFile.Close()

	var links []string

	for link := range linksSet {
		links = append(links, link)
	}

	for i, link := range links {
		wg.Add(1)
		go scrapeRokuDocsLink(link, allocatorContext, &wg)

		if i%10 == 0 {
			wg.Wait()
		}
	}
	wg.Wait()
}

func scrapeRokuDocsLink(link string, ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	fmt.Printf("Started: %s\n", link)

	ctx, cancel := chromedp.NewContext(ctx)
	defer cancel()

	var html string
	err := chromedp.Run(ctx,
		chromedp.Navigate(link),
		chromedp.WaitVisible(".markdown-body > h1:nth-child(1)"),
		chromedp.InnerHTML(".markdown-body", &html, chromedp.NodeVisible),
	)

	if err != nil {
		log.Fatal("Failed to scrape:", link, err)
		return
	}

	outputPath := fmt.Sprintf("./output/%s", strings.Split(link, "https://developer.roku.com/docs/")[1])
	outputPath = outputPath[:len(outputPath)-len(".md")] + ".html"

	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		log.Fatalf("Failed to create directory: %s", err)
	}

	outputFile, err := os.Create(outputPath)

	if err != nil {
		log.Fatalf("Failed to create file %s: %v", outputPath, err)
	}
	defer outputFile.Close()

	_, err = fmt.Fprintf(outputFile, "<!-- %s -->\n%s", link, html)
	if err != nil {
		log.Fatalf("Failed to write to file %s: %v", outputPath, err)
	}

	fmt.Printf("Finished: %s\n", link)
}
