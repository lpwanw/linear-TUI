package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/taynguyen/linear-tui/internal/views"
)

// View renders the full terminal frame as a string.
func (m Model) View() string {
	if m.width == 0 {
		return "initializing…"
	}

	header := m.renderHeader()
	status := m.renderStatus()
	footer := m.renderFooter()

	// Body height = total - header(1) - footer(1) - status(1) - optional overlay prompt(1)
	extra := 0
	if m.cmdActive || m.searchActive {
		extra = 1
	}
	body := m.renderBody(m.height - 3 - extra)
	overlay := m.renderOverlayPrompt()

	parts := []string{header, body}
	if overlay != "" {
		parts = append(parts, overlay)
	}
	parts = append(parts, footer, status)
	out := strings.Join(parts, "\n")

	if m.helpVisible {
		out = overlayCenter(out, m.renderHelp(), m.width, m.height)
	}
	if m.modal != nil {
		modalW := m.width / 2
		if modalW < 40 {
			modalW = 40
		}
		out = overlayCenter(out, m.modal.Render(modalW), m.width, m.height)
	}
	return out
}

func (m Model) renderHeader() string {
	issues := m.currentIssues()
	total := 0
	switch m.view {
	case ViewMyIssues:
		total = len(m.issuesMy)
	case ViewTriage:
		total = len(m.issuesTriage[m.selectedTeamID])
	}
	count := fmt.Sprintf(" %d/%d", len(issues), total)
	if m.searchQuery == "" && len(issues) == total {
		count = fmt.Sprintf(" %d", total)
	}
	name := string(m.view)
	if m.view == ViewTriage {
		team := m.teamKey(m.selectedTeamID)
		if team != "" {
			name = "triage · " + team
		}
	}
	return views.StyleHeader.Render(name) + views.StyleDim.Render(count)
}

func (m Model) renderBody(height int) string {
	if height < 3 {
		height = 3
	}
	listW, detailW := paneWidths(m.width)

	listBody := views.RenderIssueList(m.currentIssues(), m.stateByID, m.userByID, m.cursor, listW, height)
	listPane := views.StylePane.Width(listW).Height(height).Render(listBody)

	detail := m.detailR.Render(m.selectedIssue(), m.stateByID, m.userByID, detailW, height)
	detailPane := views.StylePane.Width(detailW).Height(height).Render(detail)

	return lipgloss.JoinHorizontal(lipgloss.Top, listPane, detailPane)
}

func (m Model) renderOverlayPrompt() string {
	if m.cmdActive {
		return m.cmdInput.View()
	}
	if m.searchActive {
		return m.searchInput.View()
	}
	return ""
}

func (m Model) renderFooter() string {
	var hints []hint
	switch {
	case m.modal != nil:
		hints = []hint{
			{"j/k", "move"}, {"enter", "confirm"}, {"esc", "cancel"},
		}
	case m.cmdActive:
		hints = []hint{
			{"enter", "run"}, {"esc", "cancel"},
			{":q", "quit"}, {":sync", "resync"}, {":view", "switch"},
			{":state|assign|priority", "mutate"}, {":open", "browser"},
		}
	case m.searchActive:
		hints = []hint{{"type", "filter"}, {"enter", "commit"}, {"esc", "clear"}}
	case m.helpVisible:
		hints = []hint{{"?/esc", "close"}}
	default:
		hints = []hint{
			{"j/k", "move"}, {"gg/G", "top/bot"}, {"1/2", "view"},
			{"s/a/p", "edit"}, {"y/Y", "yank br/url"},
			{"/", "filter"}, {":", "cmd"}, {"r", "refresh"},
			{"?", "help"}, {"q", "quit"},
		}
	}
	return renderHints(hints, m.width)
}

type hint struct{ key, label string }

func renderHints(hs []hint, width int) string {
	sep := views.StyleDim.Render(" · ")
	var out strings.Builder
	used := 0
	for i, h := range hs {
		part := views.StyleHelpKey.Render(h.key) + views.StyleDim.Render(" "+h.label)
		// Rough visible-width estimate (ignores ANSI). Keeps footer from wrapping.
		vis := len(h.key) + 1 + len(h.label)
		add := vis
		if i > 0 {
			add += 3 // sep width
		}
		if used+add > width {
			break
		}
		if i > 0 {
			out.WriteString(sep)
		}
		out.WriteString(part)
		used += add
	}
	return out.String()
}

func (m Model) renderStatus() string {
	var parts []string
	if m.syncing {
		parts = append(parts, m.spinner.View()+" syncing")
	} else if !m.lastSyncedAt.IsZero() {
		parts = append(parts, "synced "+views.RelativeTime(m.lastSyncedAt)+" ago")
	}
	if m.searchQuery != "" {
		parts = append(parts, "filter: "+m.searchQuery)
	}
	status := views.StyleStatus.Render(strings.Join(parts, " · "))
	if m.infoBanner != "" {
		status += "   " + views.StyleInfo.Render(m.infoBanner)
	}
	if m.errorBanner != "" {
		status += "   " + views.StyleError.Render(m.errorBanner)
	}
	return status
}

func (m Model) renderHelp() string {
	bindings := [][2]string{
		{"1 / 2", "switch view: my issues / triage"},
		{"j / k", "cursor down / up"},
		{"gg / G", "jump top / bottom"},
		{"Ctrl-d / u", "half-page down / up"},
		{"Ctrl-f / b", "page down / up"},
		{"s / a / p", "state / assignee / priority picker"},
		{"y / Y", "yank branch slug / url to clipboard"},
		{"/", "live filter"},
		{"r", "refresh current view"},
		{":sync", "full resync"},
		{":view my_issues|triage", "switch view"},
		{":state <name>", "change state"},
		{":assign <name|me|none>", "change assignee"},
		{":priority <n|name>", "change priority"},
		{":open", "open issue URL"},
		{":q / q", "quit"},
		{"?", "toggle this help"},
		{"esc", "close modal / cancel search / dismiss error"},
	}
	var b strings.Builder
	b.WriteString(views.StyleHeader.Render("linear-tui — keybindings"))
	b.WriteString("\n\n")
	for _, kb := range bindings {
		b.WriteString(views.StyleHelpKey.Render(pad(kb[0], 24)) + kb[1] + "\n")
	}
	b.WriteString("\n" + views.StyleDim.Render("press ? or esc to close · ")+
		"version "+time.Now().Format("2006"))
	return views.StyleModalBorder.Render(b.String())
}

func (m Model) teamKey(id string) string {
	for _, t := range m.teams {
		if t.ID == id {
			return t.Key
		}
	}
	return ""
}

func pad(s string, w int) string {
	if len(s) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-len(s))
}

func overlayCenter(base, overlay string, w, h int) string {
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, overlay, lipgloss.WithWhitespaceChars(" "))
}
