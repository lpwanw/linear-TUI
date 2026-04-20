package views

import (
	"strings"

	"github.com/taynguyen/linear-tui/internal/cache"
)

// RenderIssueList renders a list of issues with a cursor row, inside the given
// pane width and height. Returns a newline-joined string with exactly `height`
// lines (padding with blanks if too short, truncating if too long).
func RenderIssueList(issues []cache.Issue, stateByID map[string]cache.WorkflowState, cursor, width, height int) string {
	if height <= 0 {
		return ""
	}
	lines := make([]string, 0, height)

	// Column budget inside the pane (width includes padding via StylePane.Padding(0,1)).
	innerWidth := width - 2 // subtract left+right padding
	if innerWidth < 10 {
		innerWidth = 10
	}

	// Column widths: icon(1) + space + ident(8) + space + state(10) + space + updated(5) + space + title(rest)
	const (
		colIcon    = 1
		colIdent   = 8
		colState   = 10
		colUpdated = 5
		gaps       = 4 // number of single-space gaps between the 5 columns
	)
	colTitle := innerWidth - colIcon - colIdent - colState - colUpdated - gaps
	if colTitle < 10 {
		colTitle = 10
	}

	start := 0
	end := len(issues)
	if end > height {
		// Center the cursor within the visible window.
		start = cursor - height/2
		if start < 0 {
			start = 0
		}
		if start+height > end {
			start = end - height
		}
		end = start + height
	}

	for i := start; i < end; i++ {
		iss := issues[i]
		stateName := ""
		if ws, ok := stateByID[iss.StateID]; ok {
			stateName = ws.Name
		}
		row := strings.Join([]string{
			PriorityIcon(iss.Priority),
			PadRight(TruncateVisual(iss.Identifier, colIdent), colIdent),
			PadRight(TruncateVisual(stateName, colState), colState),
			PadRight(TruncateVisual(RelativeTime(iss.UpdatedAt), colUpdated), colUpdated),
			TruncateVisual(iss.Title, colTitle),
		}, " ")
		if i == cursor {
			row = StyleCursor.Render(PadRight(row, innerWidth))
		}
		lines = append(lines, row)
	}
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", innerWidth))
	}
	return strings.Join(lines, "\n")
}
