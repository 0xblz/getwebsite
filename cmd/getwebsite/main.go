package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/0xblz/getwebsite/internal/fetcher"
	"github.com/0xblz/getwebsite/internal/parser"
	"github.com/0xblz/getwebsite/internal/renderer"
	"github.com/0xblz/getwebsite/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: getwebsite <url> [--pipe] [--width N] [--export FILE]")
		os.Exit(1)
	}

	url := os.Args[1]
	pipeMode := false
	width := 90
	exportPath := ""

	for i := 2; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--pipe", "-p":
			pipeMode = true
		case "--width", "-w":
			if i+1 < len(os.Args) {
				fmt.Sscanf(os.Args[i+1], "%d", &width)
				i++
			}
		case "--export", "-e":
			if i+1 < len(os.Args) {
				exportPath = os.Args[i+1]
				i++
			}
		}
	}

	// Detect if stdout is not a terminal (piping)
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		pipeMode = true
	}

	url = fetcher.NormalizeURL(url)

	// Export and pipe modes need to fetch + parse here
	if pipeMode || exportPath != "" {
		f := fetcher.New()

		fmt.Fprintf(os.Stderr, "Fetching %s...\n", url)

		html, err := f.Fetch(url)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		article, err := parser.Parse(html, url)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing: %v\n", err)
			os.Exit(1)
		}

		if exportPath != "" {
			md := renderer.RenderMarkdown(article)
			if err := os.WriteFile(exportPath, []byte(md), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", exportPath, err)
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "Exported to %s\n", exportPath)
			if pipeMode {
				r := renderer.New(width)
				fmt.Print(r.RenderArticle(article))
			}
			return
		}

		r := renderer.New(width)
		fmt.Print(r.RenderArticle(article))
		return
	}

	// Interactive mode — UI handles fetching with spinner
	m := ui.New(url)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	if len(os.Args) > 1 {
		arg := strings.ToLower(os.Args[1])
		if arg == "--help" || arg == "-h" {
			fmt.Println("getwebsite - Read websites beautifully in your terminal")
			fmt.Println()
			fmt.Println("Usage: getwebsite <url> [options]")
			fmt.Println()
			fmt.Println("Options:")
			fmt.Println("  --pipe, -p       Output plain text (no interactive UI)")
			fmt.Println("  --width, -w N    Set output width (default: 90)")
			fmt.Println("  --export, -e F   Export article as markdown to file F")
			fmt.Println("  --help, -h       Show this help")
			fmt.Println("  --version, -v    Show version")
			fmt.Println()
			fmt.Println("Examples:")
			fmt.Println("  getwebsite example.com")
			fmt.Println("  getwebsite https://news.ycombinator.com --pipe")
			fmt.Println("  getwebsite blaze.design --width 120")
			fmt.Println("  getwebsite blaze.design --export article.md")
			os.Exit(0)
		}
		if arg == "--version" || arg == "-v" {
			fmt.Println("getwebsite v0.1.0")
			os.Exit(0)
		}
	}
}
