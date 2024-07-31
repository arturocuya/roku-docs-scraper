package main

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"roku-docs-scraper/utils"
	"strings"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/JohannesKaufmann/html-to-markdown/plugin"
)

func main() {
	err := filepath.WalkDir("./output/raw/", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".html") {
			return nil
		}

		fileContent, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		converter := md.NewConverter("", true, nil)
		converter.Use(plugin.GitHubFlavored())

		markdown, err := converter.ConvertString(string(fileContent))
		if err != nil {
			return err
		}

		utils.WriteNewFile(
			strings.Replace(
				strings.Replace(path, "output/raw", "output/md/docs", 1),
				".html", ".md", 1,
			),
			markdown,
		)

		return nil
	})

	if err != nil {
		log.Fatalf("Error reading raw file %v", err)
	}
}
