# Web to PDF Converter

A command-line tool written in Go that converts web pages to PDF files. It supports batch processing of URLs and can automatically extract URLs from an index page.

It's simply a wrapper around [chromedp](https://github.com/chromedp/chromedp) to generate PDFs.

## Features

- Convert single or multiple URLs to PDF
- Extract URLs from an index page
- Customizable PDF scaling
- Parallel processing of URLs
- Custom output directory
- Optional filename prefix
- Support for query parameters
- Landscape orientation
- Background graphics included

## Prerequisites

- Go 1.16 or higher
- Chrome/Chromium browser (required for headless PDF generation)

## Installation

```bash
go get github.com/petermunro/url2pdf
```

## Build

```bash
# Clone the repository
git clone https://github.com/petermunro/url2pdf.git
cd url2pdf

# Build the binary
go build -o url2pdf

# Optional: install to $GOPATH/bin
go install
```

## Usage

```bash
go run main.go [flags]
```

### Flags

- `-urls`: Comma-separated list of URLs to convert
- `-index`: URL of the directory index page (alternative to -urls)
- `-output`: Directory to save PDFs (default: "pdfs")
- `-scale`: Scale of the webpage rendering (between 0.1 and 2.0, default: 1.0)
- `-prefix`: Prefix to add to output filenames
- `-portrait`: Print in portrait mode (default is landscape)
- `-query`: Query parameter to append to URLs (e.g. 'print=true')

### Examples

Convert a single URL:

```bash
go run main.go -urls "https://example.com/page.json"
```

Convert multiple URLs:

```bash
go run main.go -urls "https://example.com/page1.json,https://example.com/page2.json"
```

Process URLs from an index page:

```bash
go run main.go -index "https://example.com/index" -query "print=true"
```

Customize output:

```bash
go run main.go -urls "https://example.com/page.json" -output "my-pdfs" -scale 1.2 -prefix "doc"
```


## Output

PDFs are saved to the specified output directory (default: "pdfs"). Filenames are generated from:
1. The last segment of the URL (for .json URLs)
2. MD5 hash of the full URL (for other URLs)

## Error Handling

- The program logs errors for individual URL processing failures but continues with remaining URLs
- Invalid scale values (outside 0.1-2.0) will cause the program to exit
- Missing required flags (-urls or -index) will cause the program to exit

