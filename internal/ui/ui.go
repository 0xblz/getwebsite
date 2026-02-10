package ui

import (
	"fmt"
	"strings"

	"github.com/0xblz/getwebsite/internal/parser"
	"github.com/0xblz/getwebsite/internal/renderer"
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
)

type Model struct {
	article  *parser.Article
	viewport viewport.Model
	ready    bool
	width    int
	height   int
	url      string
}

func New(article *parser.Article, url string) Model {
	return Model{
		article: article,
		url:     url,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "g":
			m.viewport.GotoTop()
			return m, nil
		case "G":
			m.viewport.GotoBottom()
			return m, nil
		}

	case tea.WindowSizeMsg:
		headerHeight := 0
		footerHeight := 1
		verticalMargin := headerHeight + footerHeight

		if !m.ready {
			m.width = msg.Width
			m.height = msg.Height
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMargin)
			m.viewport.YPosition = headerHeight

			r := renderer.New(min(msg.Width, 90))
			content := r.RenderArticle(m.article)
			m.viewport.SetContent(content)
			m.ready = true
		} else {
			m.width = msg.Width
			m.height = msg.Height
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMargin

			r := renderer.New(min(msg.Width, 90))
			content := r.RenderArticle(m.article)
			m.viewport.SetContent(content)
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	footer := m.renderFooter()
	return m.viewport.View() + "\n" + footer
}

func (m Model) renderFooter() string {
	percent := fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100)

	keys := []struct{ key, desc string }{
		{"↑/k", "up"},
		{"↓/j", "down"},
		{"g/G", "top/bottom"},
		{"q", "quit"},
	}

	var parts []string
	for _, k := range keys {
		parts = append(parts,
			helpKeyStyle.Render("["+k.key+"]")+" "+helpStyle.Render(k.desc))
	}

	help := strings.Join(parts, "  ")
	percentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	gap := m.width - lipgloss.Width(help) - lipgloss.Width(percent) - 2
	if gap < 1 {
		gap = 1
	}

	return help + strings.Repeat(" ", gap) + percentStyle.Render(percent)
}
