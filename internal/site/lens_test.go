package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// percale: a lens is references in the syntax where: uses, each with an
// optional note; every line shows, resolved or saying why not.
func TestParseLens(t *testing.T) {
	l := ParseLens("---\nrepository: abc1234\n---\n\n# Auth\n\nSign-in, for the module.\n\n- `plans/a.md › One`\n- `plans/b.md` — the whole plan\n- ``plans/c.md › Has `ticks` `` : with a note\n")
	if l.Title != "Auth" || l.Repository != "abc1234" || l.Intro != "Sign-in, for the module." || len(l.Items) != 3 {
		t.Fatalf("%+v", l)
	}
	if l.Items[1].Ref != "plans/b.md" || l.Items[1].Note != "the whole plan" || l.Items[2].Ref != "plans/c.md › Has `ticks`" || l.Items[2].Note != "with a note" {
		t.Errorf("%+v", l.Items)
	}
}

func TestLensResolvesAndShows(t *testing.T) {
	repo := &Repo{Dir: "../.."}
	s, err := BuildSite(repo, "51b503c", NewStore(repo, t.TempDir(), nil))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, LensDir), 0o755)
	os.WriteFile(filepath.Join(dir, LensDir, "verdicts.md"), []byte("---\nrepository: 0000000\n---\n\n# Verdicts\n\n- `plans/2026-10-30-burlap.md › The verdict shows on the answer and on its guess`\n- `plans/2026-10-31-organdy.md` — the rewrite\n- `plans/2026-10-31-organdy.as-built.md › A Go repository needs nothing installed but the binary and scip-go`\n- `plans/2026-10-30-burlap.md › Nothing by this title`\n- `plans/2026-10-31-organdy.md › Guess: A Go repository needs nothing installed but the binary and scip-go`\n- `plans/2026-10-31-organdy.md › Decline: A Go repository needs nothing installed but the binary and scip-go`\n"), 0o644)
	lenses := ReadLenses(s, dir)
	if len(lenses) != 1 || lenses[0].Name != "verdicts" {
		t.Fatalf("%+v", lenses)
	}
	l := lenses[0]
	if l.Items[4].Entry == nil || l.Items[5].Entry != nil {
		t.Errorf("a kind in front resolves only when it's the entry's own: %+v %+v", l.Items[4], l.Items[5])
	}
	if l.Items[0].Entry == nil || l.Items[1].Entry != nil || l.Items[1].Doc == nil || l.Items[2].Doc != l.Items[2].Plan.AsBuilt || !strings.HasPrefix(l.Items[3].Problem, "no entry titled") {
		t.Errorf("%+v", l.Items)
	}
	page := LensPage(s, l, FirstCommit(repo)).HTML
	for _, want := range []string{`itemprop="itemCount" content="6"`, `itemprop="repositoryNote"`, `rel="lens-item" itemprop="entry" href="../../plans/burlap/#the-verdict-shows-on-the-answer-and-on-its-guess"`, `<span itemprop="kind">Plan</span>`, `itemprop="problem">no entry titled this in burlap`, `itemprop="note">the rewrite`} {
		if !strings.Contains(page, want) {
			t.Errorf("no %s", want)
		}
	}
	if n := strings.Count(page, `itemprop="status">partly disproved`); n != 2 {
		t.Errorf("the guess and its answer should both say partly disproved, %d do", n)
	}
	list := LensesPage(s, lenses, FirstCommit(repo)).HTML
	if !strings.Contains(list, `itemprop="firstCommit" value="`+FirstCommit(repo)+`"`) || !strings.Contains(list, `href="verdicts/"`) {
		t.Errorf("lenses page:\n%s", list)
	}
}
