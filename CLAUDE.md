# CLAUDE.md

## Build & Test

```bash
go build ./cmd/getwebsite          # build
go vet ./...                       # lint
go build -o getwebsite ./cmd/getwebsite  # build binary
./getwebsite <url> --pipe          # quick test (non-interactive)
./getwebsite <url> --export out.md # test markdown export
```

No test suite yet — verify manually with `--pipe` mode.

## Architecture

```
cmd/getwebsite/main.go        → CLI entry point, flag parsing, orchestration
internal/fetcher/fetcher.go   → HTTP client, URL normalization
internal/parser/parser.go     → HTML → Article with typed ContentBlocks
internal/renderer/renderer.go → ContentBlocks → styled terminal output (lipgloss)
internal/renderer/images.go   → ASCII art / iTerm2 inline image rendering
internal/renderer/markdown.go → ContentBlocks → markdown export
internal/ui/ui.go             → Bubbletea TUI (viewport, spinner, search, keybindings)
```

### Key types

- `parser.Article` — title, description, site name, content blocks, links
- `parser.ContentBlock` — tagged union via `BlockType` (heading, paragraph, code, list, quote, image, table, hr)
- `renderer.Renderer` — stateful; tracks `HeadingLines` for section jumping after `RenderArticle()`
- `ui.Model` — bubbletea model; handles loading state, search, link opening, section jumping

### Data flow

- **Pipe/export mode:** main.go fetches + parses, then calls renderer directly
- **Interactive mode:** main.go passes URL to `ui.New(url)`, UI fetches in background (spinner), then renders into viewport

### Conventions

- Page metadata comes from readability `Excerpt` + `<meta>` tag fallbacks (og:description, meta description, twitter:description) — not author byline
- Images render at the bottom in a dedicated "Images" section, not inline
- Links are footnote-style `[N]` in text, collected in a "Links" section at bottom
- Tables use box-drawing characters (┌─┬─┐ etc.)
- All terminal styling uses lipgloss; colors are defined as package vars in renderer.go
