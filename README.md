Usage:

- `go run cmd/scraper/main.go` to scrape the docs into `./output/raw`
- `go run cmd/html2md/main.go` to convert the scraped html to markdown (may have problems with nested tables) into `./output/md`

I would then open the markdown files with something like [Obsidian](https://obsidian.md/).

_Not sure if this is legal, but if not open an issue :)_
