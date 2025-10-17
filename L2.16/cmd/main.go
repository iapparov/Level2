package main

import (
	"flag"
	"fmt"
	"goWget/internal/downloader"
	"goWget/internal/parser"
	"os"
	"time"
)

func main() {

	var (
		urlFlag        = flag.String("url", "", "root URL to mirror (required)")
		depthFlag      = flag.Int("depth", 2, "recursion depth")
		outFlag        = flag.String("out", "", "output directory")
		concurrency    = flag.Int("concurrency", 6, "number of parallel downloads")
		timeoutSeconds = flag.Int("timeout", 5, "HTTP client timeout in seconds")
	)
	flag.Parse()
	if *urlFlag == "" {
		fmt.Fprintln(os.Stderr, "url parameter is required. Example: -url=https://example.com -depth=2 -out=./site")
		os.Exit(2)
	}
	c, err := parser.NewCrawler(*urlFlag, *depthFlag, *concurrency)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid url: %v\n", err)
		os.Exit(1)
	}
	start := time.Now()
	fmt.Printf("Start mirror %s -> %s (depth=%d, concurrency=%d)\n", *urlFlag, *outFlag, *depthFlag, *concurrency)
	d := downloader.NewDownloader(*urlFlag, *outFlag, time.Duration(*timeoutSeconds)*time.Second)
	if err := c.Start(d); err != nil {
		fmt.Fprintf(os.Stderr, "crawl finished with error: %v\n", err)
	}
	fmt.Printf("Done in %s\n", time.Since(start))
}
