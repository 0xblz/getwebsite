package renderer

import (
	"fmt"
	"strings"

	"github.com/0xblz/getwebsite/internal/parser"
	"github.com/charmbracelet/lipgloss"
)

var (
	ColorHeading   = lipgloss.Color("205")
	ColorH2        = lipgloss.Color("212")
	ColorH3        = lipgloss.Color("218")
	ColorLink      = lipgloss.Color("86")
	ColorCode      = lipgloss.Color("228")
	ColorCodeBG    = lipgloss.Color("236")
	ColorQuote     = lipgloss.Color("243")
	ColorQuoteLine = lipgloss.Color("205")
	ColorMeta      = lipgloss.Color("243")
	ColorBullet    = lipgloss.Color("205")
	ColorImage     = lipgloss.Color("243")
	ColorHR        = lipgloss.Color("240")
)

type Renderer struct {
	width int
}

func New(width int) *Renderer {
	return &Renderer{width: width}
}

func (r *Renderer) RenderArticle(article *parser.Article) string {
	var b strings.Builder

	// Title header
	b.WriteString(r.renderTitle(article))
	b.WriteString("\n\n")

	// Content blocks
	for i, block := range article.Content {
		// Add a subtle divider before headings (except the first block)
		if block.Type == parser.BlockHeading && i > 0 {
			dividerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
			b.WriteString(dividerStyle.Render("  "+strings.Repeat("─", r.width-4)) + "\n")
		}

		rendered := r.RenderBlock(block)
		if rendered != "" {
			b.WriteString(rendered)
			b.WriteString("\n")
		}
	}

	return b.String()
}

func (r *Renderer) renderTitle(article *parser.Article) string {
	contentWidth := r.width - 4

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorHeading).
		Width(contentWidth)

	title := titleStyle.Render(article.Title)

	var meta string
	parts := []string{}
	if article.Author != "" {
		parts = append(parts, "by "+article.Author)
	}
	if article.SiteName != "" {
		parts = append(parts, article.SiteName)
	}
	if len(parts) > 0 {
		metaStyle := lipgloss.NewStyle().
			Foreground(ColorMeta).
			Width(contentWidth)
		meta = metaStyle.Render(strings.Join(parts, " · "))
	}

	content := title
	if meta != "" {
		content += "\n" + meta
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2).
		Width(r.width)

	return boxStyle.Render(content)
}

func (r *Renderer) RenderBlock(block parser.ContentBlock) string {
	switch block.Type {
	case parser.BlockHeading:
		return r.renderHeading(block)
	case parser.BlockParagraph:
		return r.renderParagraph(block)
	case parser.BlockCode:
		return r.renderCode(block)
	case parser.BlockList:
		return r.renderList(block)
	case parser.BlockQuote:
		return r.renderQuote(block)
	case parser.BlockImage:
		return r.renderImage(block)
	case parser.BlockHR:
		return r.renderHR()
	default:
		return ""
	}
}

func (r *Renderer) renderHeading(block parser.ContentBlock) string {
	color := ColorHeading
	prefix := "▸ "
	switch block.Level {
	case 1:
		color = ColorHeading
		prefix = "▸ "
	case 2:
		color = ColorH2
		prefix = "▸ "
	case 3:
		color = ColorH3
		prefix = "  ▹ "
	default:
		color = ColorH3
		prefix = "    ▹ "
	}

	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(color)

	return "\n" + style.Render(prefix+block.Text) + "\n"
}

func (r *Renderer) renderParagraph(block parser.ContentBlock) string {
	style := lipgloss.NewStyle().
		Width(r.width - 2).
		PaddingLeft(1)

	return style.Render(block.Text) + "\n"
}

func (r *Renderer) renderCode(block parser.ContentBlock) string {
	codeStyle := lipgloss.NewStyle().
		Foreground(ColorCode).
		Background(ColorCodeBG).
		Padding(0, 1)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		MarginLeft(2).
		Width(r.width - 6)

	code := codeStyle.Render(block.Text)
	return boxStyle.Render(code) + "\n"
}

func (r *Renderer) renderList(block parser.ContentBlock) string {
	var b strings.Builder
	bulletStyle := lipgloss.NewStyle().Foreground(ColorBullet)

	for i, item := range block.Items {
		var prefix string
		if block.Ordered {
			prefix = fmt.Sprintf("  %d. ", i+1)
		} else {
			prefix = "  " + bulletStyle.Render("•") + " "
		}

		itemStyle := lipgloss.NewStyle().
			Width(r.width - 6).
			PaddingLeft(0)

		b.WriteString(prefix + itemStyle.Render(item) + "\n")
	}

	return b.String()
}

func (r *Renderer) renderQuote(block parser.ContentBlock) string {
	barStyle := lipgloss.NewStyle().
		Foreground(ColorQuoteLine).
		Bold(true)

	textStyle := lipgloss.NewStyle().
		Foreground(ColorQuote).
		Italic(true).
		Width(r.width - 8).
		PaddingLeft(1)

	bar := barStyle.Render("┃")
	lines := strings.Split(textStyle.Render(block.Text), "\n")
	var b strings.Builder
	for _, line := range lines {
		b.WriteString("  " + bar + " " + line + "\n")
	}

	return b.String()
}

func (r *Renderer) renderImage(block parser.ContentBlock) string {
	style := lipgloss.NewStyle().
		Foreground(ColorImage).
		Italic(true)

	return style.Render("  [IMAGE: "+block.Alt+"]") + "\n"
}

func (r *Renderer) renderHR() string {
	style := lipgloss.NewStyle().
		Foreground(ColorHR)

	return style.Render("  " + strings.Repeat("━", r.width-4)) + "\n"
}
