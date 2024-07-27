package main

import "strings"

func IsRokuDocsUrlValid(url *string) bool {
	if *url == "" {
		return false
	}
	if !strings.HasPrefix(*url, "https://developer.roku.com/") {
		return false
	}
	if !strings.Contains(*url, "/docs/") {
		return false
	}
	return true
}

func SanitizeRokuDocsUrl(url string) string {
	splittedUrl := strings.Split(url, "/")
	if splittedUrl[3] != "docs" {
		url = strings.Join(append(splittedUrl[:3], splittedUrl[4:]...), "/")
	}

	anchorIndex := strings.Index(url, "#")

	if anchorIndex != -1 {
		url = url[:anchorIndex]
	}

	return url
}
