package site

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// A diff's hunks are read in the old side's numbering; an insertion touches
// the lines either side; a deleted file is the whole file (chambray).
func TestParseDiff(t *testing.T) {
	diff := `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -3,2 +3,4 @@ func x() {
@@ -10 +12 @@
@@ -20,0 +23,2 @@
diff --git a/gone.go b/gone.go
deleted file mode 100644
--- a/gone.go
+++ /dev/null
@@ -1,5 +0,0 @@
diff --git a/new.go b/new.go
new file mode 100644
--- /dev/null
+++ b/new.go
@@ -0,0 +1,3 @@
`
	got := ParseDiff(diff)
	want := Changed{"a.go": {{3, 4}, {10, 10}, {20, 21}}, "gone.go": nil}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ParseDiff = %v, want %v", got, want)
	}
	for _, c := range []struct {
		file  string
		lines [][2]int
		want  bool
	}{
		{"a.go", [][2]int{{4, 6}}, true},
		{"a.go", [][2]int{{5, 9}}, false},
		{"a.go", nil, true},
		{"a.go", [][2]int{}, false}, // a quote that isn't found is on no line
		{"gone.go", [][2]int{{2, 2}}, true},
		{"other.go", nil, false},
	} {
		if got := want.Touches(c.file, c.lines); got != c.want {
			t.Errorf("Touches(%s, %v) = %v, want %v", c.file, c.lines, got, c.want)
		}
	}
}

func TestWrapSpan(t *testing.T) {
	for in, want := range map[string]string{
		"plans/a.md › Plain":         "`plans/a.md › Plain`",
		"plans/a.md › `init` prints": "`` plans/a.md › `init` prints ``",
	} {
		if got := wrapSpan(in); got != want {
			t.Errorf("wrapSpan(%q) = %q, want %q", in, got, want)
		}
		if inner, _, ok := codeSpan(wrapSpan(in)); !ok || inner != in {
			t.Errorf("a lens reads %q back as %q", wrapSpan(in), inner)
		}
	}
}

// affected finds an entry by its file, and a change finds it only when it
// touches the quoted lines; the output reads back as a lens.
func TestAffectedFindsAnchoredEntries(t *testing.T) {
	dir := t.TempDir()
	git := func(args ...string) string {
		out, err := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@t"}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatal(string(out))
		}
		return strings.TrimSpace(string(out))
	}
	write := func(name, text string) {
		os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0o755)
		os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644)
	}
	git("init", "-q")
	write("src/a.txt", "one\ntwo\nthree\nfour\nfive\nsix\n")
	git("add", ".")
	git("commit", "-q", "-m", "code")
	base := git("rev-parse", "--short", "HEAD")
	write("plans/2026-01-01-alpha.md", "---\nbase: "+base+"\n---\n\n# Alpha\n\n## Decision: Four stays four\n- where: `src/a.txt`\n- quote: `four`\n\n**Because (t):** it is.\n\n## Decision: Something else\n- where: `src/b.txt`\n- quote: `x`\n\n**Because (t):** no.\n")
	git("add", ".")
	git("commit", "-q", "-m", "plan")
	repo := &Repo{Dir: dir}
	store := NewStore(repo, t.TempDir(), nil)

	s, err := BuildSite(repo, "HEAD", store)
	if err != nil {
		t.Fatal(err)
	}
	hits := Affected(s, InFiles([]string{"src/a.txt"}))
	if len(hits) != 1 || hits[0].Entry.Title != "Four stays four" || hits[0].Line != 4 {
		t.Fatalf("by file: %+v", hits)
	}
	line := LensLine(hits[0])
	if l := ParseLens("# x\n\n" + line + "\n"); len(l.Items) != 1 || l.Items[0].Ref != "plans/2026-01-01-alpha.md › Four stays four" {
		t.Errorf("%q doesn't read back as a lens item: %+v", line, l.Items)
	}

	write("src/a.txt", "one\nTWO\nthree\nfour\nfive\nsix\n")
	changed, at, err := ChangedSince(repo, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	s, _ = BuildSite(repo, at, store)
	if hits := Affected(s, changed.Touches); len(hits) != 0 {
		t.Errorf("line 2 changed, but it found %+v", hits)
	}
	write("src/a.txt", "one\ntwo\nthree\nFOUR\nfive\nsix\n")
	changed, at, _ = ChangedSince(repo, "HEAD")
	s, _ = BuildSite(repo, at, store)
	if hits := Affected(s, changed.Touches); len(hits) != 1 {
		t.Errorf("line 4 changed, found %+v", hits)
	}
}

// A quote that's gone says the last of its commits it was found at and the
// commit that removed it, with that commit's Plan: trailer, and nothing about
// what that means (madras).
func TestAGoneQuoteSaysWhatRemovedIt(t *testing.T) {
	dir := t.TempDir()
	git := func(args ...string) string {
		out, err := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@t"}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatal(string(out))
		}
		return strings.TrimSpace(string(out))
	}
	write := func(name, text string) {
		os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0o755)
		os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644)
	}
	git("init", "-q")
	write("a.txt", "one\nfour\n")
	git("add", ".")
	git("commit", "-q", "-m", "code")
	base := git("rev-parse", "--short", "HEAD")
	write("plans/2026-01-01-alpha.md", "---\nbase: "+base+"\n---\n\n# Alpha\n\n## Decision: Four stays four\n- where: `a.txt`\n- quote: `four`\n\n**Because (t):** it is.\n")
	git("add", ".")
	git("commit", "-q", "-m", "plan")
	write("a.txt", "one\nFIVE\n")
	git("commit", "-q", "-am", "beta: four becomes five\n\nPlan: plans/2026-01-02-beta.md")
	removed := git("rev-parse", "--short", "HEAD")
	repo := &Repo{Dir: dir}
	s, err := BuildSite(repo, "HEAD", NewStore(repo, t.TempDir(), nil))
	if err != nil {
		t.Fatal(err)
	}
	hits := Affected(s, InFiles([]string{"a.txt"}))
	if len(hits) != 1 {
		t.Fatalf("hits: %+v", hits)
	}
	line := LensLine(hits[0])
	for _, want := range []string{"last found at `" + base + "`", "removed in `" + removed + "`", "“beta: four becomes five”", "Plan: plans/2026-01-02-beta.md"} {
		if !strings.Contains(line, want) {
			t.Errorf("%q lacks %q", line, want)
		}
	}
	for _, verdict := range []string{"drift", "own work", "reconciled"} {
		if strings.Contains(line, verdict) {
			t.Errorf("%q says what it means (%q)", line, verdict)
		}
	}
}
