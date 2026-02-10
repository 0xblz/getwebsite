package ui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/0xblz/getwebsite/internal/fetcher"
	"github.com/0xblz/getwebsite/internal/parser"
	"github.com/0xblz/getwebsite/internal/renderer"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))
	helpKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true)
	searchHighlightStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("205")).
				Foreground(lipgloss.Color("0")).
				Bold(true)
)

// Messages
type articleMsg struct {
	article *parser.Article
	err     error
}

type Model struct {
	// Core
	article  *parser.Article
	viewport viewport.Model
	ready    bool
	width    int
	height   int
	url      string

	// Loading
	loading bool
	spinner spinner.Model

	// Search
	searching     bool
	searchInput   textinput.Model
	searchQuery   string
	searchMatches []int // line indices
	searchIdx     int   // current match index
	contentLines  []string

	// Section jumping
	headingLines []int

	// Open link
	openingLink bool
	linkInput   textinput.Model

	// Rendered content (pre-highlight)
	rawContent string
}

func New(url string) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	si := textinput.New()
	si.Prompt = "/"
	si.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	si.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	si.CharLimit = 100

	li := textinput.New()
	li.Prompt = "Open link #: "
	li.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
	li.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	li.CharLimit = 10

	return Model{
		url:         url,
		loading:     true,
		spinner:     s,
		searchInput: si,
		linkInput:   li,
	}
}

func fetchArticle(url string) tea.Cmd {
	return func() tea.Msg {
		f := fetcher.New()
		html, err := f.Fetch(url)
		if err != nil {
			return articleMsg{err: err}
		}
		article, err := parser.Parse(html, url)
		if err != nil {
			return articleMsg{err: err}
		}
		return articleMsg{article: article}
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, fetchArticle(m.url))
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case articleMsg:
		if msg.err != nil {
			// Show error and quit
			m.loading = false
			m.ready = true
			m.viewport.SetContent(fmt.Sprintf("Error: %v", msg.err))
			return m, nil
		}
		m.article = msg.article
		m.loading = false
		if m.width > 0 {
			m.renderContent()
		}
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case tea.KeyMsg:
		// Search mode input
		if m.searching {
			switch msg.String() {
			case "enter":
				query := m.searchInput.Value()
				if query != "" {
					m.searchQuery = query
					m.executeSearch()
					if len(m.searchMatches) > 0 {
						m.searchIdx = 0
						m.jumpToMatch()
					}
				}
				m.searching = false
				m.searchInput.Blur()
				return m, nil
			case "esc":
				m.searching = false
				m.searchInput.Blur()
				// Clear search highlights
				if m.rawContent != "" {
					m.viewport.SetContent(m.rawContent)
				}
				m.searchQuery = ""
				m.searchMatches = nil
				return m, nil
			default:
				var cmd tea.Cmd
				m.searchInput, cmd = m.searchInput.Update(msg)
				return m, cmd
			}
		}

		// Open link mode input
		if m.openingLink {
			switch msg.String() {
			case "enter":
				numStr := m.linkInput.Value()
				if num, err := strconv.Atoi(numStr); err == nil && m.article != nil {
					for _, link := range m.article.Links {
						if link.Index == num {
							openBrowser(link.URL)
							break
						}
					}
				}
				m.openingLink = false
				m.linkInput.Blur()
				m.linkInput.SetValue("")
				return m, nil
			case "esc":
				m.openingLink = false
				m.linkInput.Blur()
				m.linkInput.SetValue("")
				return m, nil
			default:
				var cmd tea.Cmd
				m.linkInput, cmd = m.linkInput.Update(msg)
				return m, cmd
			}
		}

		// Normal mode keys
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc":
			// Clear search if active, otherwise quit
			if m.searchQuery != "" {
				m.searchQuery = ""
				m.searchMatches = nil
				if m.rawContent != "" {
					m.viewport.SetContent(m.rawContent)
				}
				return m, nil
			}
			return m, tea.Quit
		case "g":
			m.viewport.GotoTop()
			return m, nil
		case "G":
			m.viewport.GotoBottom()
			return m, nil
		case "/":
			if !m.loading {
				m.searching = true
				m.searchInput.SetValue("")
				m.searchInput.Focus()
				return m, textinput.Blink
			}
		case "n":
			if len(m.searchMatches) > 0 {
				m.searchIdx = (m.searchIdx + 1) % len(m.searchMatches)
				m.jumpToMatch()
			}
			return m, nil
		case "N":
			if len(m.searchMatches) > 0 {
				m.searchIdx = (m.searchIdx - 1 + len(m.searchMatches)) % len(m.searchMatches)
				m.jumpToMatch()
			}
			return m, nil
		case "]":
			m.jumpToNextHeading()
			return m, nil
		case "[":
			m.jumpToPrevHeading()
			return m, nil
		case "o":
			if !m.loading && m.article != nil && len(m.article.Links) > 0 {
				m.openingLink = true
				m.linkInput.SetValue("")
				m.linkInput.Focus()
				return m, textinput.Blink
			}
		}

	case tea.WindowSizeMsg:
		headerHeight := 0
		footerHeight := 1
		verticalMargin := headerHeight + footerHeight

		m.width = msg.Width
		m.height = msg.Height

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMargin)
			m.viewport.YPosition = headerHeight
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMargin
		}

		if !m.loading && m.article != nil {
			m.renderContent()
		}
	}

	if m.ready && !m.loading {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) renderContent() {
	r := renderer.New(min(m.width, 90))
	content := r.RenderArticle(m.article)
	m.rawContent = content
	m.headingLines = r.HeadingLines
	m.contentLines = strings.Split(content, "\n")

	// Re-apply search highlights if search is active
	if m.searchQuery != "" {
		m.executeSearch()
		m.applyHighlights()
	} else {
		m.viewport.SetContent(content)
	}
}

func (m *Model) executeSearch() {
	m.searchMatches = nil
	if m.searchQuery == "" {
		return
	}
	query := strings.ToLower(m.searchQuery)
	for i, line := range m.contentLines {
		if strings.Contains(strings.ToLower(line), query) {
			m.searchMatches = append(m.searchMatches, i)
		}
	}
	m.applyHighlights()
}

func (m *Model) applyHighlights() {
	if len(m.searchMatches) == 0 {
		m.viewport.SetContent(m.rawContent)
		return
	}

	matchSet := make(map[int]bool)
	for _, idx := range m.searchMatches {
		matchSet[idx] = true
	}

	var highlighted strings.Builder
	for i, line := range m.contentLines {
		if matchSet[i] {
			highlighted.WriteString(searchHighlightStyle.Render(line))
		} else {
			highlighted.WriteString(line)
		}
		if i < len(m.contentLines)-1 {
			highlighted.WriteString("\n")
		}
	}
	m.viewport.SetContent(highlighted.String())
}

func (m *Model) jumpToMatch() {
	if m.searchIdx < len(m.searchMatches) {
		line := m.searchMatches[m.searchIdx]
		m.viewport.SetYOffset(line)
	}
}

func (m *Model) jumpToNextHeading() {
	if len(m.headingLines) == 0 {
		return
	}
	current := m.viewport.YOffset
	for _, line := range m.headingLines {
		if line > current {
			m.viewport.SetYOffset(line)
			return
		}
	}
	// Wrap around to first heading
	m.viewport.SetYOffset(m.headingLines[0])
}

func (m *Model) jumpToPrevHeading() {
	if len(m.headingLines) == 0 {
		return
	}
	current := m.viewport.YOffset
	for i := len(m.headingLines) - 1; i >= 0; i-- {
		if m.headingLines[i] < current {
			m.viewport.SetYOffset(m.headingLines[i])
			return
		}
	}
	// Wrap around to last heading
	m.viewport.SetYOffset(m.headingLines[len(m.headingLines)-1])
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return
	}
	_ = cmd.Start()
}

func (m Model) View() string {
	if m.loading {
		if !m.ready {
			return ""
		}
		loadingStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true)
		urlStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("86"))
		msg := m.spinner.View() + " " + loadingStyle.Render("Fetching") + " " + urlStyle.Render(m.url) + loadingStyle.Render("...")
		// Center vertically
		padding := m.height / 3
		return strings.Repeat("\n", padding) + "  " + msg
	}

	if !m.ready {
		return ""
	}

	footer := m.renderFooter()
	return m.viewport.View() + "\n" + footer
}

func (m Model) renderFooter() string {
	// Search mode footer
	if m.searching {
		return m.searchInput.View()
	}

	// Open link mode footer
	if m.openingLink {
		return m.linkInput.View()
	}

	percent := fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100)

	keys := []struct{ key, desc string }{
		{"↑/k", "up"},
		{"↓/j", "down"},
		{"/", "search"},
		{"n/N", "next/prev"},
		{"]/[", "sections"},
		{"o", "open link"},
		{"q", "quit"},
	}

	var parts []string
	for _, k := range keys {
		parts = append(parts,
			helpKeyStyle.Render("["+k.key+"]")+" "+helpStyle.Render(k.desc))
	}

	help := strings.Join(parts, "  ")

	// If search is active, show match info
	if m.searchQuery != "" {
		matchInfo := helpStyle.Render(fmt.Sprintf("  [%d/%d matches]", m.searchIdx+1, len(m.searchMatches)))
		if len(m.searchMatches) == 0 {
			matchInfo = helpStyle.Render("  [no matches]")
		}
		help += matchInfo
	}

	percentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	gap := m.width - lipgloss.Width(help) - lipgloss.Width(percent) - 2
	if gap < 1 {
		gap = 1
	}

	return help + strings.Repeat(" ", gap) + percentStyle.Render(percent)
}
