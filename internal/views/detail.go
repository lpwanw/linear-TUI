package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"

	"github.com/taynguyen/linear-tui/internal/cache"
)

// DetailRenderer wraps a glamour TermRenderer configured for the current pane
// width. Rebuild on width changes.
type DetailRenderer struct {
	width    int
	gl       *glamour.TermRenderer
	mdCache  map[string]string // issue.ID -> rendered markdown
}

func NewDetailRenderer(width int) *DetailRenderer {
	dr := &DetailRenderer{width: width, mdCache: map[string]string{}}
	dr.reinitGlamour()
	return dr
}

func (d *DetailRenderer) SetWidth(w int) {
	if w == d.width || w <= 0 {
		return
	}
	d.width = w
	d.mdCache = map[string]string{}
	d.reinitGlamour()
}

func (d *DetailRenderer) Invalidate(issueID string) {
	delete(d.mdCache, issueID)
}

func (d *DetailRenderer) reinitGlamour() {
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(d.width-4),
	)
	if err != nil {
		d.gl = nil
		return
	}
	d.gl = r
}

// Render returns a multiline string describing the issue. width/height bound it.
func (d *DetailRenderer) Render(issue *cache.Issue, stateByID map[string]cache.WorkflowState, userByID map[string]cache.User, width, height int) string {
	if issue == nil {
		return StyleDim.Render("No issue selected")
	}
	d.SetWidth(width)

	var lines []string
	lines = append(lines, StyleDetailTitle.Render(fmt.Sprintf("%s %s", issue.Identifier, issue.Title)))
	lines = append(lines, "")

	state := stateByID[issue.StateID].Name
	assignee := "(unassigned)"
	if u, ok := userByID[issue.AssigneeID]; ok && issue.AssigneeID != "" {
		assignee = u.Name
	}
	meta := fmt.Sprintf("state: %s · priority: %s · assignee: %s", state, cache.PriorityLabel(issue.Priority), assignee)
	lines = append(lines, StyleDim.Render(meta))
	if issue.URL != "" {
		lines = append(lines, StyleDim.Render("url:  "+issue.URL))
	}
	lines = append(lines, "")

	desc := issue.Description
	if strings.TrimSpace(desc) == "" {
		lines = append(lines, StyleDim.Render("(no description)"))
	} else {
		rendered, ok := d.mdCache[issue.ID]
		if !ok && d.gl != nil {
			r, err := d.gl.Render(desc)
			if err == nil {
				rendered = strings.TrimRight(r, "\n")
				d.mdCache[issue.ID] = rendered
				ok = true
			}
		}
		if !ok {
			rendered = desc
		}
		lines = append(lines, strings.Split(rendered, "\n")...)
	}

	if height > 0 && len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\n")
}
