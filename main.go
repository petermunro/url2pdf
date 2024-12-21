package main

import (
	"context"
	"crypto/md5"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

func main() {
	// Define command line flags
	outputDir := flag.String("output", "pdfs", "Directory to save PDFs")
	urls := flag.String("urls", "", "Comma-separated list of URLs to convert")
	scale := flag.Float64("scale", 1.0, "Scale of the webpage rendering (between 0.1 and 2.0)")
	indexURL := flag.String("index", "", "URL of the directory index page")
	prefix := flag.String("prefix", "", "Prefix to add to output filenames")
	queryParam := flag.String("query", "", "Query parameter to append to URLs (e.g. 'print=true')")
	portrait := flag.Bool("portrait", false, "Print in portrait mode (default is landscape)")
	flag.Parse()

	if *urls == "" && *indexURL == "" {
		log.Fatal("Please provide either -urls or -index flag")
	}

	// Validate scale
	if *scale < 0.1 || *scale > 2.0 {
		log.Fatal("Scale must be between 0.1 and 2.0")
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	var urlList []string
	if *indexURL != "" {
		var err error
		urlList, err = getURLsFromIndexPage(*indexURL, *queryParam)
		if err != nil {
			log.Fatalf("Failed to process index page: %v", err)
		}
	} else {
		// Process URLs from -urls flag
		if *urls != "" {
			urlList = strings.Split(*urls, ",")
			if *queryParam != "" {
				for i, url := range urlList {
					if strings.Contains(url, "?") {
						urlList[i] = url + "&" + *queryParam
					} else {
						urlList[i] = url + "?" + *queryParam
					}
				}
			}
		}
	}

	// Create a wait group to track goroutines
	var wg sync.WaitGroup

	// Create a single browser instance
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.DisableGPU,
		chromedp.NoSandbox,
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	browserCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Process each URL
	for _, url := range urlList {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			if err := generatePDF(browserCtx, url, *outputDir, *scale, *prefix, *portrait); err != nil {
				log.Printf("Error processing %s: %v", url, err)
			}
		}(strings.TrimSpace(url))
	}

	wg.Wait()
}

func generatePDF(ctx context.Context, url, outputDir string, scale float64, prefix string, portrait bool) error {
	// Create a new tab for each URL
	tabCtx, cancel := chromedp.NewContext(ctx)
	defer cancel()

	// Generate filename from URL
	filename := generateFilename(url, outputDir, prefix)

	var pdf []byte
	if err := chromedp.Run(tabCtx,
		chromedp.Navigate(url),
		chromedp.WaitReady("body"),
		chromedp.WaitVisible(`body`, chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			pdf, _, err = page.PrintToPDF().
				WithPrintBackground(true).
				WithLandscape(!portrait).
				WithScale(scale).
				Do(ctx)
			return err
		}),
	); err != nil {
		return fmt.Errorf("failed to generate PDF: %v", err)
	}

	// Save PDF to file
	if err := os.WriteFile(filename, pdf, 0644); err != nil {
		return fmt.Errorf("failed to save PDF: %v", err)
	}

	fmt.Printf("Successfully generated PDF for %s: %s\n", url, filename)
	return nil
}

func generateFilename(url, outputDir, prefix string) string {
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		lastPart := parts[len(parts)-1]
		if queryIndex := strings.Index(lastPart, "?"); queryIndex != -1 {
			lastPart = lastPart[:queryIndex]
		}
		if strings.HasSuffix(lastPart, ".json") {
			baseName := strings.TrimSuffix(lastPart, ".json")
			if prefix != "" {
				baseName = prefix + "-" + baseName
			}
			return filepath.Join(outputDir, baseName+".pdf")
		}
	}
	// Fallback case
	h := md5.New()
	io.WriteString(h, url)
	return filepath.Join(outputDir, fmt.Sprintf("%x.pdf", h.Sum(nil)))
}

func getURLsFromIndexPage(indexURL string, queryParam string) ([]string, error) {
	fmt.Printf("Loading page: %s\n", indexURL)

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	var links []string
	err := chromedp.Run(ctx,
		chromedp.Navigate(indexURL),
		chromedp.Sleep(2*time.Second), // Wait for React to render
		chromedp.Evaluate(`
			Array.from(document.querySelectorAll('a'))
				.filter(a => a.href.endsWith('.json'))
				.map(a => a.href)
		`, &links),
	)

	if queryParam != "" {
		for i, link := range links {
			if strings.Contains(link, "?") {
				links[i] = link + "&" + queryParam
			} else {
				links[i] = link + "?" + queryParam
			}
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to extract links: %v", err)
	}

	fmt.Printf("Found %d JSON URLs\n", len(links))
	for _, link := range links {
		fmt.Printf("  %s\n", link)
	}

	return links, nil
}
