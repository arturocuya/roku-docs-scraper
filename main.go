package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

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
	var pageUrls []string

	for _, targetUrl := range targetUrls {
		wg.Add(1)
		go func(targetUrl TargetUrl) {
			defer wg.Done()

			fmt.Printf("Started scrapping main url: %s\n", targetUrl.Url)

			ctx, cancel := chromedp.NewContext(allocatorContext)
			defer cancel()

			var urls []string
			err := chromedp.Run(ctx,
				chromedp.Navigate(targetUrl.Url),
				chromedp.WaitVisible(targetUrl.SelectorToWaitVisible),
				chromedp.Evaluate(`Array.from(document.querySelectorAll('a')).map(a => a.href)`, &urls),
			)
			if err != nil {
				fmt.Println("Failed to scrape:", targetUrl.Url, err)
				return
			}

			mux.Lock()
			// TODO: Is the TargetUrl.Url also included in pageUrls?
			pageUrls = append(pageUrls, urls...)
			mux.Unlock()

			fmt.Printf("Finished scrapping main url: %s\n", targetUrl.Url)
		}(targetUrl)
	}

	wg.Wait()

	urlSet := make(map[string]struct{})
	var exists = struct{}{}

	for _, url := range pageUrls {
		if IsRokuDocsUrlValid(&url) {
			urlSet[SanitizeRokuDocsUrl(url)] = exists
		}
	}

	var urls []string

	for uniqueUrl := range urlSet {
		urls = append(urls, uniqueUrl)
	}

	for i, url := range urls {
		wg.Add(1)
		go scrapeRokuDocsUrl(url, allocatorContext, &wg)

		if i%10 == 0 {
			wg.Wait()
		}
	}
	wg.Wait()
}

func scrapeRokuDocsUrl(url string, allocatorContext context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	fmt.Printf("Started: %s\n", url)

	ctx, cancel := chromedp.NewContext(allocatorContext)
	defer cancel()

	var html string
	var containerClasses string

	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitVisible(".markdown-body"),
		chromedp.Sleep(1*time.Second),
		chromedp.AttributeValue(".content > div:nth-child(2)", "class", &containerClasses, nil),
	)

	if err != nil {
		log.Fatal("Failed to scrape:", url, err)
		return
	}

	if strings.Contains(containerClasses, "doc-error") {
		html = "Content container has .doc-error class. Scrapping aborted."
	} else {
		err := chromedp.Run(ctx,
			chromedp.WaitVisible(".markdown-body > h1:nth-child(1)"),
			chromedp.InnerHTML(".markdown-body", &html, chromedp.NodeVisible),
		)
		if err != nil {
			log.Fatal("Failed to scrape:", url, err)
			return
		}
	}

	outputPath := fmt.Sprintf("./output/%s", strings.Split(url, "https://developer.roku.com/docs/")[1])
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

	_, err = fmt.Fprintf(outputFile, "<!-- %s -->\n%s", url, html)
	if err != nil {
		log.Fatalf("Failed to write to file %s: %v", outputPath, err)
	}

	fmt.Printf("Finished: %s\n", url)
}
