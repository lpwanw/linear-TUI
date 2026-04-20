package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/taynguyen/linear-tui/internal/cache"
)

// Style registry. Flat for now; Phase 4 promotes to a Theme struct.
var (
	StyleCursor      = lipgloss.NewStyle().Bold(true).Reverse(true)
	StyleIdentifier  = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	StyleTitle       = lipgloss.NewStyle()
	StyleStateBadge  = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	StyleUpdatedAt   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	StylePriUrgent   = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	StylePriHigh     = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	StylePriNormal   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	StylePriLow      = lipgloss.NewStyle().Foreground(lipgloss.Color("242"))
	StylePriNone     = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	StyleHeader      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	StyleStatus      = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	StyleError       = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	StyleDim         = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	StylePane        = lipgloss.NewStyle().Padding(0, 1)
	StyleDetailTitle = lipgloss.NewStyle().Bold(true)
	StyleModalBorder = lipgloss.NewStyle().Border(lipgloss.RoundedBorder(), true).Padding(0, 1)
	StyleModalTitle  = lipgloss.NewStyle().Bold(true).Underline(true)
	StyleHelpKey     = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
)

func PriorityIcon(p int) string {
	switch p {
	case cache.PriorityUrgent:
		return StylePriUrgent.Render("●")
	case cache.PriorityHigh:
		return StylePriHigh.Render("●")
	case cache.PriorityNormal:
		return StylePriNormal.Render("●")
	case cache.PriorityLow:
		return StylePriLow.Render("●")
	default:
		return StylePriNone.Render("·")
	}
}

// RelativeTime returns a short "3m", "5h", "2d" string.
func RelativeTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	default:
		return t.Format("2006-01-02")
	}
}

// TruncateVisual truncates a string to width cells (rune-aware, not byte).
// For full grapheme correctness we'd use uniseg, but this is good enough for
// issue titles, which rarely contain emoji or combining marks.
func TruncateVisual(s string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= width {
		return s
	}
	if width < 2 {
		return string(runes[:width])
	}
	return string(runes[:width-1]) + "…"
}

// PadRight pads s with spaces to width cells.
func PadRight(s string, width int) string {
	r := []rune(s)
	if len(r) >= width {
		return string(r)
	}
	return s + strings.Repeat(" ", width-len(r))
}
