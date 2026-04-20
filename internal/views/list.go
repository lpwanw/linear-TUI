package views

import (
	"fmt"
	"strings"

	"github.com/taynguyen/linear-tui/internal/cache"
)

// RenderIssueList renders issues with a cursor row, inside the given pane
// width and height. The cursor row is 3 lines tall (primary + 2 metadata);
// other rows are 1 line. Returns a newline-joined string padded or truncated
// to exactly `height` lines.
func RenderIssueList(issues []cache.Issue, stateByID map[string]cache.WorkflowState, userByID map[string]cache.User, cursor, width, height int) string {
	if height <= 0 {
		return ""
	}
	innerWidth := width - 2
	if innerWidth < 10 {
		innerWidth = 10
	}

	const (
		colIcon     = 1
		colIdent    = 8
		colState    = 10
		colUpdated  = 5
		gaps        = 4
		cursorRowH  = 3
	)
	colTitle := innerWidth - colIcon - colIdent - colState - colUpdated - gaps
	if colTitle < 10 {
		colTitle = 10
	}

	n := len(issues)
	start, end := visibleWindow(cursor, n, height, cursorRowH)

	var lines []string
	for i := start; i < end; i++ {
		iss := issues[i]
		stateName := ""
		if ws, ok := stateByID[iss.StateID]; ok {
			stateName = ws.Name
		}
		primary := strings.Join([]string{
			PriorityIcon(iss.Priority),
			PadRight(TruncateVisual(iss.Identifier, colIdent), colIdent),
			PadRight(TruncateVisual(stateName, colState), colState),
			PadRight(TruncateVisual(RelativeTime(iss.UpdatedAt), colUpdated), colUpdated),
			TruncateVisual(iss.Title, colTitle),
		}, " ")

		if i == cursor {
			lines = append(lines, cursorRowLines(iss, userByID, primary, innerWidth)...)
			if len(lines) >= height {
				break
			}
		} else {
			lines = append(lines, primary)
		}
	}

	// Pad or truncate to exactly `height` lines.
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", innerWidth))
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\n")
}

// cursorRowLines returns the 3 lines of a highlighted selected row.
func cursorRowLines(iss cache.Issue, userByID map[string]cache.User, primary string, innerWidth int) []string {
	assignee := "(unassigned)"
	if u, ok := userByID[iss.AssigneeID]; ok && iss.AssigneeID != "" {
		assignee = u.Name
	}
	meta := fmt.Sprintf("  %s priority · %s", cache.PriorityLabel(iss.Priority), assignee)
	detail := "  " + iss.Title
	return []string{
		StyleCursor.Render(PadRight(TruncateVisual(primary, innerWidth), innerWidth)),
		StyleCursor.Render(PadRight(TruncateVisual(meta, innerWidth), innerWidth)),
		StyleCursor.Render(PadRight(TruncateVisual(detail, innerWidth), innerWidth)),
	}
}

// visibleWindow picks [start, end) such that cursor is on-screen, reserving
// extra lines for the expanded cursor row.
func visibleWindow(cursor, n, height, cursorRowH int) (int, int) {
	if n == 0 {
		return 0, 0
	}
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= n {
		cursor = n - 1
	}
	// Effective capacity in rows treating cursor as cursorRowH-1 extra rows.
	cap := height - (cursorRowH - 1)
	if cap < 1 {
		cap = 1
	}
	if n <= cap {
		return 0, n
	}
	start := cursor - cap/2
	if start < 0 {
		start = 0
	}
	end := start + cap
	if end > n {
		end = n
		start = end - cap
		if start < 0 {
			start = 0
		}
	}
	return start, end
}
