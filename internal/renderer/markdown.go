package renderer

import (
	"fmt"
	"strings"

	"github.com/0xblz/getwebsite/internal/parser"
)

// RenderMarkdown converts an Article back to markdown.
func RenderMarkdown(article *parser.Article) string {
	var b strings.Builder

	// Title
	b.WriteString("# " + article.Title + "\n\n")

	// Metadata
	var meta []string
	if article.Author != "" {
		meta = append(meta, "by "+article.Author)
	}
	if article.SiteName != "" {
		meta = append(meta, article.SiteName)
	}
	if len(meta) > 0 {
		b.WriteString("*" + strings.Join(meta, " · ") + "*\n\n")
	}

	b.WriteString("---\n\n")

	for _, block := range article.Content {
		switch block.Type {
		case parser.BlockHeading:
			b.WriteString(strings.Repeat("#", block.Level) + " " + block.Text + "\n\n")

		case parser.BlockParagraph:
			b.WriteString(block.Text + "\n\n")

		case parser.BlockCode:
			lang := block.Language
			b.WriteString("```" + lang + "\n")
			b.WriteString(block.Text + "\n")
			b.WriteString("```\n\n")

		case parser.BlockList:
			for i, item := range block.Items {
				if block.Ordered {
					b.WriteString(fmt.Sprintf("%d. %s\n", i+1, item))
				} else {
					b.WriteString("- " + item + "\n")
				}
			}
			b.WriteString("\n")

		case parser.BlockQuote:
			lines := strings.Split(block.Text, "\n")
			for _, line := range lines {
				b.WriteString("> " + line + "\n")
			}
			b.WriteString("\n")

		case parser.BlockImage:
			alt := block.Alt
			if alt == "" {
				alt = "image"
			}
			b.WriteString(fmt.Sprintf("![%s](%s)\n\n", alt, block.URL))

		case parser.BlockTable:
			if len(block.Rows) == 0 {
				continue
			}
			numCols := 0
			for _, row := range block.Rows {
				if len(row) > numCols {
					numCols = len(row)
				}
			}
			// Write first row
			if len(block.Rows) > 0 {
				b.WriteString("|")
				for j := 0; j < numCols; j++ {
					cell := ""
					if j < len(block.Rows[0]) {
						cell = block.Rows[0][j]
					}
					b.WriteString(" " + cell + " |")
				}
				b.WriteString("\n")
				// Separator line (required for valid markdown tables)
				b.WriteString("|")
				for j := 0; j < numCols; j++ {
					b.WriteString(" --- |")
				}
				b.WriteString("\n")
			}
			// Remaining rows
			for _, row := range block.Rows[1:] {
				b.WriteString("|")
				for j := 0; j < numCols; j++ {
					cell := ""
					if j < len(row) {
						cell = row[j]
					}
					b.WriteString(" " + cell + " |")
				}
				b.WriteString("\n")
			}
			b.WriteString("\n")

		case parser.BlockHR:
			b.WriteString("---\n\n")
		}
	}

	// Links
	if len(article.Links) > 0 {
		b.WriteString("---\n\n")
		b.WriteString("## Links\n\n")
		for _, link := range article.Links {
			b.WriteString(fmt.Sprintf("[%d]: %s (%s)\n", link.Index, link.URL, link.Text))
		}
	}

	return b.String()
}
