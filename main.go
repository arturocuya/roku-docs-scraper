package main

import (
	"context"
	"fmt"
	"log"
	"os"
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

	for link := range linksSet {
		if link != "" && strings.HasPrefix(link, "https://developer.roku.com") {
			_, err := fmt.Fprintln(outputFile, link)
			if err != nil {
				log.Fatalf("Failed to write to file: %v", err)
			}
		}
	}
}
