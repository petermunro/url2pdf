package main

import (
	"context"
	"flag"
	"fmt"
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
	flag.Parse()

	if *urls == "" {
		log.Fatal("Please provide at least one URL using the -urls flag")
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Split URLs into slice
	urlList := strings.Split(*urls, ",")

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
			if err := generatePDF(browserCtx, url, *outputDir); err != nil {
				log.Printf("Error processing %s: %v", url, err)
			}
		}(strings.TrimSpace(url))
	}

	wg.Wait()
}

func generatePDF(ctx context.Context, url, outputDir string) error {
	// Create a new tab for each URL
	tabCtx, cancel := chromedp.NewContext(ctx)
	defer cancel()

	// Generate filename from URL
	filename := generateFilename(url, outputDir)

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
				WithLandscape(true).
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

func generateFilename(url, outputDir string) string {
	// Remove protocol prefix and replace special characters
	name := strings.TrimPrefix(url, "http://")
	name = strings.TrimPrefix(name, "https://")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, ":", "_")
	name = strings.ReplaceAll(name, "?", "_")
	name = strings.ReplaceAll(name, "&", "_")
	name = strings.ReplaceAll(name, "=", "_")

	// Ensure filename ends with .pdf
	if !strings.HasSuffix(name, ".pdf") {
		name += ".pdf"
	}

	return filepath.Join(outputDir, name)
}
