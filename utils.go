package main

import "strings"

func IsRokuDocsURLValid(url *string) bool {
	if *url == "" {
		return false
	}
	if !strings.HasPrefix(*url, "https://developer.roku.com/") {
		return false
	}
	// ASSUMPTION: All documentation URLs contain a /docs/ substring
	if !strings.Contains(*url, "/docs/") {
		return false
	}
	return true
}

func SanitizeRokuDocsURL(url string) string {
	/*
		ASSUMPTION: When you split docs URLs by "/", you can have two different
		values in the third position:
		1. The word "docs"
		2. An ISO language code, like "en-gb"
		The latter should be omitted from the URL
	*/
	splittedURL := strings.Split(url, "/")

	if splittedURL[3] != "docs" {
		url = strings.Join(append(splittedURL[:3], splittedURL[4:]...), "/")
	}

	anchorIndex := strings.Index(url, "#")

	if anchorIndex != -1 {
		url = url[:anchorIndex]
	}

	return url
}
