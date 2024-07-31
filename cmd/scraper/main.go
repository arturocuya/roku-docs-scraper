package main

import (
	"context"
	"fmt"
	"log"
	"roku-docs-scraper/utils"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
)

const MaxChromiumInstances = 10

type TargetURL struct {
	URL                   string
	SelectorToWaitVisible string
}

func NewTargetURL(url, selectorToWaitVisible string) *TargetURL {
	return &TargetURL{
		URL:                   url,
		SelectorToWaitVisible: selectorToWaitVisible,
	}
}

func main() {
	// ASSUMPTION: The following URLs are the main sections of the Roku documentation,
	// each with a different element that indicates when the page is ready to extract
	// the links from it.
	targetURLs := []TargetURL{
		*NewTargetURL(
			"https://developer.roku.com/docs/features/features-overview.md",
			"#document-nav-menu > nav:nth-child(1) > div:nth-child(2) > div:nth-child(1) > ul:nth-child(1) > li:nth-child(1) > a:nth-child(1)",
		),
		*NewTargetURL(
			"https://developer.roku.com/docs/specs/specs-overview.md",
			"#document-nav-menu > nav:nth-child(1) > div:nth-child(2) > div:nth-child(1) > ul:nth-child(1) > li:nth-child(1) > a:nth-child(1)",
		),
		*NewTargetURL(
			"https://developer.roku.com/docs/developer-program/getting-started/roku-dev-prog.md",
			"#document-nav-menu > nav:nth-child(1) > div:nth-child(2) > div:nth-child(1) > ul:nth-child(1) > li:nth-child(1) > div:nth-child(2) > ul:nth-child(1) > li:nth-child(1)",
		),
		*NewTargetURL(
			"https://developer.roku.com/docs/references/references-overview.md",
			"#document-nav-menu > nav:nth-child(1) > div:nth-child(2) > div:nth-child(1) > ul:nth-child(1) > li:nth-child(1) > a:nth-child(1)",
		),
	}

	allocatorContext, cancel := chromedp.NewExecAllocator(context.Background(), append(
		chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
	)...)
	defer cancel()

	var wg sync.WaitGroup
	var mux sync.Mutex
	var pageURLs []string

	for _, targetURL := range targetURLs {
		wg.Add(1)
		go func(targetURL TargetURL) {
			defer wg.Done()

			fmt.Printf("Started scrapping main url: %s\n", targetURL.URL)

			ctx, cancel := chromedp.NewContext(allocatorContext)
			defer cancel()

			var URLs []string
			err := chromedp.Run(ctx,
				chromedp.Navigate(targetURL.URL),
				chromedp.WaitVisible(targetURL.SelectorToWaitVisible),
				chromedp.Evaluate(
					`Array.from(document.querySelectorAll('a')).map(a => a.href)`,
					&URLs),
			)
			if err != nil {
				fmt.Println("Failed to scrape main url:", targetURL.URL, err)
				return
			}

			mux.Lock()
			pageURLs = append(pageURLs, URLs...)
			mux.Unlock()

			fmt.Printf("Finished scrapping main url: %s\n", targetURL.URL)
		}(targetURL)
	}

	wg.Wait()

	// The keys of a map with empty values serves as a set
	urlSet := make(map[string]struct{})
	var exists = struct{}{}

	for _, url := range pageURLs {
		if utils.IsRokuDocsURLValid(&url) {
			urlSet[utils.SanitizeRokuDocsURL(url)] = exists
		}
	}

	var finalURLs []string

	for uniqueURL := range urlSet {
		finalURLs = append(finalURLs, uniqueURL)
	}

	for i, url := range finalURLs {
		wg.Add(1)
		go scrapeRokuDocsURL(url, allocatorContext, &wg)

		if i%MaxChromiumInstances == 0 {
			wg.Wait()
		}
	}
	wg.Wait()
}

func scrapeRokuDocsURL(url string, allocatorContext context.Context, wg *sync.WaitGroup) {
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

	outputPath := fmt.Sprintf(
		"./output/raw/%s",
		strings.Split(url, "https://developer.roku.com/docs/")[1])

	utils.WriteNewFile(outputPath, fmt.Sprintf("<!-- %s -->\n%s", url, html))

	fmt.Printf("Finished: %s\n", url)
}
