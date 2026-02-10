package parser

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	readability "github.com/go-shiori/go-readability"
)

type Article struct {
	Title       string
	Author      string
	SiteName    string
	PublishDate time.Time
	Content     []ContentBlock
	Links       []Link
	RawHTML     string
}

type Link struct {
	Index int
	Text  string
	URL   string
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
	URL      string   // image URL
}

func Parse(rawHTML []byte, pageURL string) (*Article, error) {
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
		article.Title = pageURL
	}

	base, _ := url.Parse(pageURL)
	article.Content, article.Links = parseHTML(doc.Content, base)

	return article, nil
}

func parseHTML(html string, base *url.URL) ([]ContentBlock, []Link) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		text := stripTags(html)
		if text != "" {
			return []ContentBlock{{Type: BlockParagraph, Text: text}}, nil
		}
		return nil, nil
	}

	ctx := &parseContext{base: base}
	doc.Find("body").Children().Each(func(_ int, s *goquery.Selection) {
		ctx.extractBlocks(s)
	})
	return ctx.blocks, ctx.links
}

type parseContext struct {
	blocks  []ContentBlock
	links   []Link
	linkIdx int
	base    *url.URL
}

func (ctx *parseContext) resolveURL(href string) string {
	if ctx.base == nil {
		return href
	}
	parsed, err := url.Parse(href)
	if err != nil {
		return href
	}
	return ctx.base.ResolveReference(parsed).String()
}

func (ctx *parseContext) extractBlocks(s *goquery.Selection) {
	tagName := goquery.NodeName(s)

	switch {
	case tagName == "h1" || tagName == "h2" || tagName == "h3" ||
		tagName == "h4" || tagName == "h5" || tagName == "h6":
		level := int(tagName[1] - '0')
		text := cleanText(s.Text())
		if text != "" {
			ctx.blocks = append(ctx.blocks, ContentBlock{
				Type:  BlockHeading,
				Level: level,
				Text:  text,
			})
		}

	case tagName == "p":
		text := ctx.extractTextWithLinks(s)
		if text != "" {
			ctx.blocks = append(ctx.blocks, ContentBlock{
				Type: BlockParagraph,
				Text: text,
			})
		}

	case tagName == "pre":
		code := s.Find("code")
		var text string
		if code.Length() > 0 {
			text = code.Text()
		} else {
			text = s.Text()
		}
		text = strings.TrimRight(text, "\n\t ")
		if text != "" {
			lang, _ := code.Attr("class")
			lang = strings.TrimPrefix(lang, "language-")
			ctx.blocks = append(ctx.blocks, ContentBlock{
				Type:     BlockCode,
				Text:     text,
				Language: lang,
			})
		}

	case tagName == "ul" || tagName == "ol":
		var items []string
		s.Find("li").Each(func(_ int, li *goquery.Selection) {
			text := ctx.extractTextWithLinks(li)
			if text != "" {
				items = append(items, text)
			}
		})
		if len(items) > 0 {
			ctx.blocks = append(ctx.blocks, ContentBlock{
				Type:    BlockList,
				Items:   items,
				Ordered: tagName == "ol",
			})
		}

	case tagName == "blockquote":
		text := ctx.extractTextWithLinks(s)
		if text != "" {
			ctx.blocks = append(ctx.blocks, ContentBlock{
				Type: BlockQuote,
				Text: text,
			})
		}

	case tagName == "hr":
		ctx.blocks = append(ctx.blocks, ContentBlock{Type: BlockHR})

	case tagName == "figure":
		img := s.Find("img")
		if img.Length() > 0 {
			alt, _ := img.Attr("alt")
			src, _ := img.Attr("src")
			ctx.blocks = append(ctx.blocks, ContentBlock{
				Type: BlockImage,
				Alt:  alt,
				URL:  ctx.resolveURL(src),
			})
		}
		caption := s.Find("figcaption")
		if caption.Length() > 0 {
			text := cleanText(caption.Text())
			if text != "" {
				ctx.blocks = append(ctx.blocks, ContentBlock{
					Type: BlockParagraph,
					Text: text,
				})
			}
		}

	case tagName == "img":
		alt, _ := s.Attr("alt")
		src, _ := s.Attr("src")
		ctx.blocks = append(ctx.blocks, ContentBlock{
			Type: BlockImage,
			Alt:  alt,
			URL:  ctx.resolveURL(src),
		})

	case tagName == "div" || tagName == "section" || tagName == "article" || tagName == "main":
		s.Children().Each(func(_ int, child *goquery.Selection) {
			ctx.extractBlocks(child)
		})

	default:
		text := ctx.extractTextWithLinks(s)
		if text != "" {
			ctx.blocks = append(ctx.blocks, ContentBlock{
				Type: BlockParagraph,
				Text: text,
			})
		}
	}
}

// extractTextWithLinks walks the DOM tree and replaces <a> tags with
// "link text [N]" where N is a footnote index, collecting the URL.
func (ctx *parseContext) extractTextWithLinks(s *goquery.Selection) string {
	var b strings.Builder
	s.Contents().Each(func(_ int, child *goquery.Selection) {
		if goquery.NodeName(child) == "a" {
			href, exists := child.Attr("href")
			text := cleanText(child.Text())
			if text == "" {
				return
			}
			if exists && href != "" && href != "#" {
				ctx.linkIdx++
				ctx.links = append(ctx.links, Link{
					Index: ctx.linkIdx,
					Text:  text,
					URL:   ctx.resolveURL(href),
				})
				b.WriteString(text)
				b.WriteString(fmt.Sprintf(" [%d]", ctx.linkIdx))
			} else {
				b.WriteString(text)
			}
		} else if goquery.NodeName(child) == "#text" {
			b.WriteString(child.Text())
		} else {
			// Recurse into other inline elements (em, strong, span, etc.)
			b.WriteString(ctx.extractTextWithLinks(child))
		}
	})
	result := b.String()
	fields := strings.Fields(decodeEntities(result))
	return strings.Join(fields, " ")
}

func cleanText(s string) string {
	s = decodeEntities(s)
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
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
	return strings.TrimSpace(result.String())
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
		"&mdash;", "\u2014",
		"&ndash;", "\u2013",
		"&hellip;", "\u2026",
		"&laquo;", "\u00ab",
		"&raquo;", "\u00bb",
		"&ldquo;", "\u201c",
		"&rdquo;", "\u201d",
		"&lsquo;", "\u2018",
		"&rsquo;", "\u2019",
	)
	return replacer.Replace(s)
}
