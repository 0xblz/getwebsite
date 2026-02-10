package parser

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	readability "github.com/go-shiori/go-readability"
)

type Article struct {
	Title       string
	Author      string
	SiteName    string
	PublishDate time.Time
	Content     []ContentBlock
	RawHTML     string
}

type BlockType int

const (
	BlockHeading BlockType = iota
	BlockParagraph
	BlockCode
	BlockList
	BlockQuote
	BlockImage
	BlockHR
)

type ContentBlock struct {
	Type     BlockType
	Text     string
	Level    int      // heading level (1-6)
	Language string   // code language
	Items    []string // list items
	Ordered  bool     // ordered list
	Alt      string   // image alt text
}

func Parse(rawHTML []byte, url string) (*Article, error) {
	reader := bytes.NewReader(rawHTML)
	doc, err := readability.FromReader(reader, nil)
	if err != nil {
		return nil, fmt.Errorf("extracting article: %w", err)
	}

	article := &Article{
		Title:    doc.Title,
		Author:   doc.Byline,
		SiteName: doc.SiteName,
		RawHTML:  doc.Content,
	}

	if article.Title == "" {
		article.Title = url
	}

	article.Content = parseHTML(doc.Content)

	return article, nil
}

func parseHTML(html string) []ContentBlock {
	var blocks []ContentBlock

	// Use a simple state-machine parser to extract blocks from readability HTML.
	// The readability output is relatively clean, so we parse the common tags.
	type tagState struct {
		tag   string
		attrs string
	}

	lines := strings.Split(html, "\n")
	var currentBlock *ContentBlock
	var listItems []string
	inList := false
	inOrderedList := false
	inPre := false
	var preContent strings.Builder

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		switch {
		case strings.HasPrefix(trimmed, "<pre"):
			inPre = true
			preContent.Reset()
			// Extract content after the tag on the same line
			after := extractAfterTag(trimmed, "pre")
			after = extractAfterTag(after, "code")
			if after != "" {
				preContent.WriteString(after)
			}

		case inPre:
			if strings.Contains(trimmed, "</pre>") {
				before := extractBeforeTag(trimmed, "/pre")
				before = extractBeforeTag(before, "/code")
				if before != "" {
					if preContent.Len() > 0 {
						preContent.WriteString("\n")
					}
					preContent.WriteString(before)
				}
				blocks = append(blocks, ContentBlock{
					Type: BlockCode,
					Text: cleanHTML(preContent.String()),
				})
				inPre = false
			} else {
				if preContent.Len() > 0 {
					preContent.WriteString("\n")
				}
				preContent.WriteString(trimmed)
			}

		case strings.HasPrefix(trimmed, "<h1"), strings.HasPrefix(trimmed, "<h2"),
			strings.HasPrefix(trimmed, "<h3"), strings.HasPrefix(trimmed, "<h4"),
			strings.HasPrefix(trimmed, "<h5"), strings.HasPrefix(trimmed, "<h6"):
			level := int(trimmed[2] - '0')
			text := stripTags(trimmed)
			if text != "" {
				blocks = append(blocks, ContentBlock{
					Type:  BlockHeading,
					Level: level,
					Text:  text,
				})
			}

		case strings.HasPrefix(trimmed, "<ul"):
			inList = true
			inOrderedList = false
			listItems = nil

		case strings.HasPrefix(trimmed, "<ol"):
			inList = true
			inOrderedList = true
			listItems = nil

		case strings.HasPrefix(trimmed, "</ul>") || strings.HasPrefix(trimmed, "</ol>"):
			if inList && len(listItems) > 0 {
				blocks = append(blocks, ContentBlock{
					Type:    BlockList,
					Items:   listItems,
					Ordered: inOrderedList,
				})
			}
			inList = false
			listItems = nil

		case inList && strings.HasPrefix(trimmed, "<li"):
			text := stripTags(trimmed)
			if text != "" {
				listItems = append(listItems, text)
			}

		case strings.HasPrefix(trimmed, "<blockquote"):
			currentBlock = &ContentBlock{Type: BlockQuote}

		case strings.HasPrefix(trimmed, "</blockquote>"):
			if currentBlock != nil && currentBlock.Text != "" {
				blocks = append(blocks, *currentBlock)
			}
			currentBlock = nil

		case strings.HasPrefix(trimmed, "<hr"):
			blocks = append(blocks, ContentBlock{Type: BlockHR})

		case strings.HasPrefix(trimmed, "<img"):
			alt := extractAttr(trimmed, "alt")
			if alt != "" {
				blocks = append(blocks, ContentBlock{
					Type: BlockImage,
					Alt:  alt,
				})
			}

		case strings.HasPrefix(trimmed, "<figure"):
			// skip figure wrapper

		case strings.HasPrefix(trimmed, "<figcaption"):
			text := stripTags(trimmed)
			if text != "" {
				blocks = append(blocks, ContentBlock{
					Type: BlockParagraph,
					Text: "  " + text,
				})
			}

		default:
			text := stripTags(trimmed)
			if text == "" {
				continue
			}
			if currentBlock != nil {
				// Inside a blockquote
				if currentBlock.Text != "" {
					currentBlock.Text += " "
				}
				currentBlock.Text += text
			} else {
				blocks = append(blocks, ContentBlock{
					Type: BlockParagraph,
					Text: text,
				})
			}
		}
	}

	return blocks
}

func stripTags(s string) string {
	var result strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(r)
		}
	}
	return decodeEntities(strings.TrimSpace(result.String()))
}

func cleanHTML(s string) string {
	s = stripTags(s)
	return s
}

func extractAfterTag(s, tag string) string {
	idx := strings.Index(s, ">")
	if idx == -1 {
		return ""
	}
	return strings.TrimSpace(s[idx+1:])
}

func extractBeforeTag(s, tag string) string {
	idx := strings.Index(s, "<")
	if idx == -1 {
		return s
	}
	return strings.TrimSpace(s[:idx])
}

func extractAttr(s, attr string) string {
	key := attr + `="`
	idx := strings.Index(s, key)
	if idx == -1 {
		key = attr + `='`
		idx = strings.Index(s, key)
		if idx == -1 {
			return ""
		}
	}
	start := idx + len(key)
	end := strings.IndexByte(s[start:], s[idx+len(attr)+1])
	if end == -1 {
		return ""
	}
	return s[start : start+end]
}

func decodeEntities(s string) string {
	replacer := strings.NewReplacer(
		"&amp;", "&",
		"&lt;", "<",
		"&gt;", ">",
		"&quot;", `"`,
		"&#39;", "'",
		"&apos;", "'",
		"&#x27;", "'",
		"&nbsp;", " ",
		"&#160;", " ",
		"&mdash;", "—",
		"&ndash;", "–",
		"&hellip;", "…",
		"&laquo;", "«",
		"&raquo;", "»",
		"&ldquo;", "\u201c",
		"&rdquo;", "\u201d",
		"&lsquo;", "\u2018",
		"&rsquo;", "\u2019",
	)
	return replacer.Replace(s)
}
