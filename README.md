# getwebsite

A terminal-based website reader built with Go and the [Charm](https://github.com/charmbracelet) TUI stack. Fetches web pages and renders them as beautifully formatted, readable terminal layouts with ANSI colors and clean typography.

## Screenshot

```
╭──────────────────────────────────────────────────────────────╮
│                                                              │
│  How to Build Great Terminal UIs                             │
│  by John Doe · Example Blog                                 │
│  3 min read · 842 words                                     │
│                                                              │
╰──────────────────────────────────────────────────────────────╯

 Terminal user interfaces are making a comeback. Here's why
 developers are choosing the terminal over web UIs for their
 tools.

▸ Why Terminal UIs Matter

 The terminal offers several advantages:

  • Fast startup time
  • Low resource usage
  • Keyboard-first navigation
  • Works over SSH

╭─────────────────────────────────────╮
│ package main                        │
│                                     │
│ func main() {                       │
│     fmt.Println("Hello, terminal!") │
│ }                                   │
╰─────────────────────────────────────╯

  ┌──────────┬─────────┬────────┐
  │ Feature  │ Status  │ Notes  │
  ├──────────┼─────────┼────────┤
  │ Tables   │ Done    │ Auto   │
  │ Search   │ Done    │ /key   │
  └──────────┴─────────┴────────┘

  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

[↑/k] up  [↓/j] down  [/] search  []/[] sections  [o] open link  [q] quit  100%
```

## Installation

### One-liner (recommended)

```bash
curl -sSL https://raw.githubusercontent.com/0xblz/getwebsite/main/install.sh | bash
```

### From source

```bash
git clone https://github.com/0xblz/getwebsite.git
cd getwebsite
go build -o getwebsite ./cmd/getwebsite
sudo mv getwebsite /usr/local/bin/
```

## Usage

```bash
# Basic usage (interactive scrollable view)
getwebsite blaze.design

# Auto-adds https:// if missing
getwebsite example.com

# Pipe mode (non-interactive, plain output)
getwebsite blaze.design --pipe

# Custom width
getwebsite blaze.design --width 120

# Export article as markdown
getwebsite blaze.design --export article.md
```

## Controls

| Key | Action |
|-----|--------|
| `↓` / `j` | Scroll down |
| `↑` / `k` | Scroll up |
| `g` | Jump to top |
| `G` | Jump to bottom |
| `/` | Search — type query, press Enter |
| `n` / `N` | Jump to next / previous search match |
| `]` / `[` | Jump to next / previous section heading |
| `o` | Open link — type link number, press Enter |
| `Esc` | Clear search / cancel input / quit |
| `q` / `Ctrl+C` | Quit |

## Project Structure

```
getwebsite/
├── cmd/
│   └── getwebsite/
│       └── main.go              # Entry point, CLI flag parsing
├── internal/
│   ├── fetcher/
│   │   └── fetcher.go           # HTTP client, URL normalization
│   ├── parser/
│   │   └── parser.go            # HTML parsing, content extraction
│   ├── renderer/
│   │   ├── renderer.go          # Lipgloss styling, terminal layout
│   │   ├── images.go            # Image rendering (ASCII/iTerm2)
│   │   └── markdown.go          # Markdown export
│   └── ui/
│       └── ui.go                # Bubbletea interactive viewport
├── install.sh                   # One-line installer script
├── .goreleaser.yml              # Cross-platform release builds
├── go.mod
└── go.sum
```

## Architecture

### Data Flow

```
URL → Fetch HTML → Extract Content → Parse to Blocks → Style with Lipgloss → Render in Bubbletea
```

### Core Components

**Fetcher** (`internal/fetcher`)
- HTTP client with 15s timeout
- URL normalization (auto-adds `https://`)
- Browser-like User-Agent header

**Parser** (`internal/parser`)
- Uses [go-readability](https://github.com/go-shiori/go-readability) for article extraction
- Strips ads, nav bars, footers, popups
- Parses into typed content blocks: headings, paragraphs, code, lists, quotes, images, tables, HRs
- HTML entity decoding

**Renderer** (`internal/renderer`)
- Lipgloss-styled output with ANSI colors
- Bordered title box with author/site metadata and reading time
- Color-coded headings, styled bullet lists, bordered code blocks
- Table rendering with box-drawing characters
- Blockquotes with colored left border
- Configurable width (default: 90 chars)
- Markdown export for offline reading

**UI** (`internal/ui`)
- Bubbletea interactive scrollable viewport
- Loading spinner while fetching
- Vim-style keybindings
- In-page search with match highlighting
- Section jumping between headings
- Open links in browser by number
- Scroll percentage indicator
- Alt-screen mode
- Auto-detects piped output and falls back to plain mode

### Display Modes

| Mode | Trigger | Description |
|------|---------|-------------|
| Interactive | Default (terminal) | Scrollable view with keybindings |
| Pipe | `--pipe` or piped stdout | Plain text output for scripting |
| Export | `--export FILE` | Save article as markdown |

## Dependencies

- [github.com/charmbracelet/bubbletea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [github.com/charmbracelet/lipgloss](https://github.com/charmbracelet/lipgloss) - Terminal styling
- [github.com/charmbracelet/bubbles](https://github.com/charmbracelet/bubbles) - TUI components (viewport, spinner, textinput)
- [github.com/go-shiori/go-readability](https://github.com/go-shiori/go-readability) - Article content extraction

## License

MIT
