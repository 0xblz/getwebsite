package renderer

import (
	"fmt"
	"strings"

	"github.com/0xblz/getwebsite/internal/parser"
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/lipgloss"
)

var (
	ColorHeading   = lipgloss.Color("205")
	ColorH2        = lipgloss.Color("212")
	ColorH3        = lipgloss.Color("218")
	ColorLink      = lipgloss.Color("86")
	ColorLinkRef   = lipgloss.Color("243")
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
	width       int
	inlineImages bool
}

func New(width int) *Renderer {
	return &Renderer{
		width:       width,
		inlineImages: supportsInlineImages(),
	}
}

func (r *Renderer) RenderArticle(article *parser.Article) string {
	var b strings.Builder

	// Title header
	b.WriteString(r.renderTitle(article))
	b.WriteString("\n\n")

	// Collect and render all images at the top
	var hasImages bool
	for _, block := range article.Content {
		if block.Type == parser.BlockImage && block.URL != "" {
			rendered := r.renderImage(block)
			if rendered != "" {
				b.WriteString(rendered)
				hasImages = true
			}
		}
	}
	if hasImages {
		dividerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
		b.WriteString(dividerStyle.Render("  "+strings.Repeat("─", r.width-4)) + "\n\n")
	}

	// Content blocks (skip images since they're already at the top)
	for i, block := range article.Content {
		if block.Type == parser.BlockImage {
			continue
		}

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

	// Link footnotes
	if len(article.Links) > 0 {
		b.WriteString(r.renderLinks(article.Links))
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
	// Colorize link references [N] within the text
	text := colorizeLinks(block.Text)

	style := lipgloss.NewStyle().
		Width(r.width - 2).
		PaddingLeft(1)

	return style.Render(text) + "\n"
}

func (r *Renderer) renderCode(block parser.ContentBlock) string {
	highlighted := highlightCode(block.Text, block.Language)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		MarginLeft(2).
		Width(r.width - 6)

	return boxStyle.Render(highlighted) + "\n"
}

func (r *Renderer) renderList(block parser.ContentBlock) string {
	var b strings.Builder
	bulletStyle := lipgloss.NewStyle().Foreground(ColorBullet)

	for i, item := range block.Items {
		item = colorizeLinks(item)
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
	lines := strings.Split(textStyle.Render(colorizeLinks(block.Text)), "\n")
	var b strings.Builder
	for _, line := range lines {
		b.WriteString("  " + bar + " " + line + "\n")
	}

	return b.String()
}

func (r *Renderer) renderImage(block parser.ContentBlock) string {
	if block.URL != "" {
		// Try iTerm2 inline image first
		if r.inlineImages {
			img := renderInlineImage(block.URL, r.width-4)
			if img != "" {
				var b strings.Builder
				b.WriteString("  " + img + "\n")
				if block.Alt != "" {
					captionStyle := lipgloss.NewStyle().
						Foreground(ColorImage).
						Italic(true)
					b.WriteString(captionStyle.Render("  "+block.Alt) + "\n")
				}
				return b.String()
			}
		}

		// Fallback to ASCII art
		ascii := renderASCIIImage(block.URL, r.width-4)
		if ascii != "" {
			var b strings.Builder
			b.WriteString(ascii)
			if block.Alt != "" {
				captionStyle := lipgloss.NewStyle().
					Foreground(ColorImage).
					Italic(true)
				b.WriteString(captionStyle.Render("  "+block.Alt) + "\n")
			}
			return b.String()
		}
	}

	// Final fallback to text placeholder
	alt := block.Alt
	if alt == "" {
		alt = "image"
	}
	style := lipgloss.NewStyle().
		Foreground(ColorImage).
		Italic(true)

	return style.Render("  [IMAGE: "+alt+"]") + "\n"
}

func (r *Renderer) renderHR() string {
	style := lipgloss.NewStyle().
		Foreground(ColorHR)

	return style.Render("  " + strings.Repeat("━", r.width-4)) + "\n"
}

func (r *Renderer) renderLinks(links []parser.Link) string {
	var b strings.Builder

	dividerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	b.WriteString("\n" + dividerStyle.Render("  "+strings.Repeat("─", r.width-4)) + "\n")

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorMeta)
	b.WriteString(headerStyle.Render("  Links") + "\n\n")

	idxStyle := lipgloss.NewStyle().Foreground(ColorLink).Bold(true)
	urlStyle := lipgloss.NewStyle().Foreground(ColorLink)
	textStyle := lipgloss.NewStyle().Foreground(ColorMeta)

	for _, link := range links {
		idx := idxStyle.Render(fmt.Sprintf("  [%d]", link.Index))
		text := textStyle.Render(link.Text)
		url := urlStyle.Render(link.URL)

		// Use OSC 8 hyperlink if possible (clickable in supported terminals)
		clickableURL := fmt.Sprintf("\033]8;;%s\033\\%s\033]8;;\033\\", link.URL, url)

		b.WriteString(fmt.Sprintf("%s %s\n      %s\n", idx, text, clickableURL))
	}

	return b.String()
}

// colorizeLinks applies styling to [N] link references within text.
func colorizeLinks(text string) string {
	refStyle := lipgloss.NewStyle().
		Foreground(ColorLink).
		Bold(true)

	var result strings.Builder
	i := 0
	for i < len(text) {
		if text[i] == '[' {
			// Look for a closing bracket with only digits inside
			j := i + 1
			for j < len(text) && text[j] >= '0' && text[j] <= '9' {
				j++
			}
			if j > i+1 && j < len(text) && text[j] == ']' {
				ref := text[i : j+1]
				result.WriteString(refStyle.Render(ref))
				i = j + 1
				continue
			}
		}
		result.WriteByte(text[i])
		i++
	}
	return result.String()
}

// highlightCode uses chroma to syntax-highlight code.
func highlightCode(code, language string) string {
	var lexer chroma.Lexer
	if language != "" {
		lexer = lexers.Get(language)
	}
	if lexer == nil {
		lexer = lexers.Analyse(code)
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	style := styles.Get("monokai")
	if style == nil {
		style = styles.Fallback
	}

	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		// Fallback: return unstyled
		codeStyle := lipgloss.NewStyle().
			Foreground(ColorCode)
		return codeStyle.Render(code)
	}

	var b strings.Builder
	err = formatter.Format(&b, style, iterator)
	if err != nil {
		codeStyle := lipgloss.NewStyle().
			Foreground(ColorCode)
		return codeStyle.Render(code)
	}

	return strings.TrimRight(b.String(), "\n")
}
