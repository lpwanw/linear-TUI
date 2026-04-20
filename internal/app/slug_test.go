package app

import "testing"

func TestBranchSlug(t *testing.T) {
	cases := []struct {
		ident, title, want string
	}{
		{"ENG-42", "Fix null pointer in sync", "eng-42-fix-null-pointer-in-sync"},
		{"BUT-1", "Add OAuth2 / PKCE support!", "but-1-add-oauth2-pkce-support"},
		{"ENG-99", "   Leading   and   trailing   ", "eng-99-leading-and-trailing"},
		{"ENG-0", "", "eng-0"},
		{"ENG-1", "émojis are 🎉 dropped", "eng-1-mojis-are-dropped"},
		{"ENG-2", strings_repeat("word ", 30), "eng-2-word-word-word-word-word-word-word-word-word-word-word-word"[:60]},
	}
	for _, c := range cases {
		got := BranchSlug(c.ident, c.title)
		if got != c.want {
			t.Errorf("BranchSlug(%q, %q) = %q, want %q", c.ident, c.title, got, c.want)
		}
	}
}

// Tiny helper to keep test table readable without importing strings twice.
func strings_repeat(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}
