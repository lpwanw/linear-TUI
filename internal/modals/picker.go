package modals

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Kind identifies which mutation a picker drives.
type Kind int

const (
	KindNone Kind = iota
	KindState
	KindAssignee
	KindPriority
)

type Item struct {
	ID    string
	Label string
	Value any
}

type Picker struct {
	Kind   Kind
	Title  string
	Items  []Item
	Cursor int
}

func (p *Picker) Down() {
	if len(p.Items) == 0 {
		return
	}
	p.Cursor = (p.Cursor + 1) % len(p.Items)
}

func (p *Picker) Up() {
	if len(p.Items) == 0 {
		return
	}
	p.Cursor = (p.Cursor - 1 + len(p.Items)) % len(p.Items)
}

func (p *Picker) Selected() *Item {
	if p.Cursor < 0 || p.Cursor >= len(p.Items) {
		return nil
	}
	return &p.Items[p.Cursor]
}

var (
	styleTitle  = lipgloss.NewStyle().Bold(true).Underline(true)
	styleBorder = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	styleCursor = lipgloss.NewStyle().Bold(true).Reverse(true)
	styleDim    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

// Render returns a formatted modal string ready for lipgloss.Place.
func (p *Picker) Render(width int) string {
	var body strings.Builder
	body.WriteString(styleTitle.Render(p.Title) + "\n\n")
	if len(p.Items) == 0 {
		body.WriteString(styleDim.Render("(no options)"))
	}
	for i, it := range p.Items {
		line := it.Label
		if i == p.Cursor {
			line = styleCursor.Render("› " + it.Label)
		} else {
			line = "  " + line
		}
		body.WriteString(line + "\n")
	}
	body.WriteString("\n" + styleDim.Render("enter=confirm  esc=cancel  j/k=move"))
	return styleBorder.Width(width).Render(body.String())
}
