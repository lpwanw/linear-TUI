package app

import (
	"os/exec"
	"runtime"
	"strings"
	"unicode"
)

// BranchSlug returns a git-branch-safe slug in the form
// "<identifier-lower>-<title-slug>", truncated to 60 chars max.
func BranchSlug(identifier, title string) string {
	parts := []string{normalizeSlug(identifier)}
	if t := normalizeSlug(title); t != "" {
		parts = append(parts, t)
	}
	slug := strings.Trim(strings.Join(parts, "-"), "-")
	if len(slug) > 60 {
		slug = strings.TrimRight(slug[:60], "-")
	}
	return slug
}

func normalizeSlug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	lastDash := false
	for _, r := range s {
		switch {
		case unicode.IsLetter(r) && r < 128, unicode.IsDigit(r) && r < 128:
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash && b.Len() > 0 {
				b.WriteRune('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

// Clipboard writes s to the system clipboard via pbcopy (macOS), wl-copy or
// xclip (Linux). Returns the tool used or an error.
func Clipboard(s string) (string, error) {
	var cmd *exec.Cmd
	var tool string
	switch runtime.GOOS {
	case "darwin":
		tool = "pbcopy"
		cmd = exec.Command("pbcopy")
	default:
		if _, err := exec.LookPath("wl-copy"); err == nil {
			tool = "wl-copy"
			cmd = exec.Command("wl-copy")
		} else if _, err := exec.LookPath("xclip"); err == nil {
			tool = "xclip"
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else if _, err := exec.LookPath("xsel"); err == nil {
			tool = "xsel"
			cmd = exec.Command("xsel", "--clipboard", "--input")
		} else {
			return "", exec.ErrNotFound
		}
	}
	cmd.Stdin = strings.NewReader(s)
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return tool, nil
}
