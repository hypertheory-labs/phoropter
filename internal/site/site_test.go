package site

import "testing"

// An anchor's text becomes a page's address only when every system can
// write it (dupioni, from the trial's `*.lens files in the workspace`).
func TestPathSafe(t *testing.T) {
	for p, want := range map[string]bool{
		"src/main.ts": true, "plans/2026-11-19-dupioni.md": true, ".github/workflows/ci.yml": true,
		"*.lens files in the workspace": false, "a/../b": false, "/etc/passwd": false, `src\main.ts`: false,
		"docs/aux.md": false, "notes./x": false, "a:b": false, "List<T>": false, "": false,
	} {
		if got := pathSafe(p); got != want {
			t.Errorf("pathSafe(%q) = %v, want %v", p, got, want)
		}
	}
}
