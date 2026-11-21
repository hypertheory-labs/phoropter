package site

// organdy: the Go version builds the same sites the Node version did,
// compared by their microdata, not their bytes. Structure must be identical;
// only prose properties (a reason, an intro, a note's comment, an answer) may
// differ, where goldmark renders an edge case differently, and those are
// listed rather than failed.

import (
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"

	"golang.org/x/net/html"

	"github.com/hypertheory-labs/phoropter/tui/microdata"
)

var prose = map[string]bool{"reason": true, "intro": true, "comment": true, "answer": true}

// dump flattens a page's microdata into sorted lines: every item's id and
// properties, with values squashed, and the page's links and title.
func dump(t *testing.T, page string, rel string) (structure, proseLines []string) {
	root, err := html.Parse(strings.NewReader(page))
	if err != nil {
		t.Fatal(err)
	}
	base, _ := url.Parse("http://site/" + rel)
	doc := microdata.Parse(root, base)
	structure = append(structure, "title = "+microdata.Squash(doc.Title))
	for _, l := range doc.Links {
		structure = append(structure, fmt.Sprintf("link %s -> %s", strings.Join(l.Rel, " "), l.Href))
	}
	var walk func(prefix string, it *microdata.Item)
	walk = func(prefix string, it *microdata.Item) {
		if it.ID != "" {
			structure = append(structure, prefix+" #"+it.ID)
		}
		names := make([]string, 0, len(it.Props))
		for n := range it.Props {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			for i, v := range it.Props[n] {
				key := fmt.Sprintf("%s.%s[%d]", prefix, n, i)
				if v.Item != nil {
					walk(key, v.Item)
					continue
				}
				line := key + " = " + microdata.Squash(v.Text)
				if prose[n] {
					proseLines = append(proseLines, line)
				} else {
					structure = append(structure, line)
				}
			}
		}
	}
	for i, it := range doc.Items {
		walk(fmt.Sprintf("item%d", i), it)
	}
	return structure, proseLines
}

func pagesIn(t *testing.T, dir string) map[string]string {
	pages := map[string]string{}
	filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasSuffix(p, ".html") {
			rel, _ := filepath.Rel(dir, p)
			b, _ := os.ReadFile(p)
			pages[filepath.ToSlash(rel)] = string(b)
		}
		return nil
	})
	return pages
}

// compare reports structural differences as failures and prose differences
// as a log, page by page.
func compare(t *testing.T, golden, got map[string]string, skip func(line string) bool) {
	t.Helper()
	for rel := range golden {
		if _, ok := got[rel]; !ok {
			t.Errorf("page %s: in the golden site, not in Go's", rel)
		}
	}
	for rel := range got {
		if _, ok := golden[rel]; !ok {
			t.Errorf("page %s: in Go's site, not in the golden one", rel)
		}
	}
	proseDiffs := 0
	for rel, g := range golden {
		h, ok := got[rel]
		if !ok {
			continue
		}
		gs, gp := dump(t, g, rel)
		hs, hp := dump(t, h, rel)
		if d := diff(gs, hs, skip); d != "" {
			t.Errorf("page %s: structure differs:\n%s", rel, d)
		}
		if d := diff(gp, hp, skip); d != "" {
			proseDiffs++
			t.Logf("page %s: prose differs (allowed):\n%s", rel, d)
		}
	}
	t.Logf("%d pages compared; %d with prose differences", len(golden), proseDiffs)
}

// lists are the pages that hold, since calico, what the Node version's home
// page listed: every plan at plans/, every file and symbol at files/ and
// symbols/.
var lists = []string{"plans/index.html", "files/index.html", "symbols/index.html"}

// homeMoved is calico's deliberate change to the golden sites: the golden home
// page's plan and place items are, exactly, the items on Go's three lists.
// Both home pages and the lists are then left out of the page-by-page
// comparison, since the golden site has no lists and its home page is gone.
func homeMoved(t *testing.T, golden, got map[string]string) {
	t.Helper()
	g := listed(t, golden["index.html"], "index.html")
	var h []string
	for _, rel := range lists {
		page, ok := got[rel]
		if !ok {
			t.Errorf("page %s: not in Go's site", rel)
			continue
		}
		h = append(h, listed(t, page, rel)...)
	}
	if d := diff(g, h, nil); d != "" {
		t.Errorf("the golden home page's items aren't the lists' items:\n%s", d)
	}
	delete(golden, "index.html")
	delete(got, "index.html")
	for _, rel := range lists {
		delete(got, rel)
	}
}

// listed is each plan and place item on a page, flattened to one line.
func listed(t *testing.T, page, rel string) []string {
	root, err := html.Parse(strings.NewReader(page))
	if err != nil {
		t.Fatal(err)
	}
	base, _ := url.Parse("http://site/" + rel)
	var out []string
	for _, it := range microdata.Parse(root, base).Items {
		for _, name := range []string{"plan", "place"} {
			for _, v := range it.Props[name] {
				if v.Item != nil {
					out = append(out, name+" "+flat(v.Item))
				}
			}
		}
	}
	return out
}

func flat(it *microdata.Item) string {
	names := make([]string, 0, len(it.Props))
	for n := range it.Props {
		names = append(names, n)
	}
	sort.Strings(names)
	var parts []string
	for _, n := range names {
		for _, v := range it.Props[n] {
			if v.Item != nil {
				parts = append(parts, n+"{"+flat(v.Item)+"}")
			} else {
				parts = append(parts, n+"="+microdata.Squash(v.Text))
			}
		}
	}
	return strings.Join(parts, " ")
}

func diff(a, b []string, skip func(string) bool) string {
	count := map[string]int{}
	for _, l := range a {
		if skip == nil || !skip(l) {
			count[l]++
		}
	}
	for _, l := range b {
		if skip == nil || !skip(l) {
			count[l]--
		}
	}
	var out []string
	for l, n := range count {
		switch {
		case n > 0:
			out = append(out, "  - "+l)
		case n < 0:
			out = append(out, "  + "+l)
		}
	}
	sort.Strings(out)
	if len(out) > 30 {
		out = append(out[:30], fmt.Sprintf("  … and %d more", len(out)-30))
	}
	return strings.Join(out, "\n")
}

func TestMain(m *testing.M) {
	// The test binary runs from a temp directory, so it's told where the
	// first-party indexers are, and caches indexes in a temp directory too.
	FirstParty, _ = filepath.Abs("../../indexers")
	cache, _ := os.MkdirTemp("", "phoropter-cache-")
	os.Setenv("XDG_CACHE_HOME", cache)
	code := m.Run()
	os.RemoveAll(cache)
	os.Exit(code)
}

var testGuide = Guide{Markdown: mustRead("../../guide.md"), Version: "commit test"}

func mustRead(p string) string {
	b, _ := os.ReadFile(p)
	return string(b)
}

func buildGolden(t *testing.T, repoDir, ref string) map[string]string {
	t.Helper()
	repo := &Repo{Dir: repoDir}
	out := t.TempDir()
	store := NewStore(repo, t.TempDir(), nil)
	if _, _, err := Build(repo, ref, out, store, testGuide); err != nil {
		t.Fatal(err)
	}
	return pagesIn(t, out)
}

const mockControl = "../../../mock-control"

func TestGoldenMockControl(t *testing.T) {
	if _, err := os.Stat(mockControl); err != nil {
		t.Skip("no ../mock-control")
	}
	golden, got := pagesIn(t, "../../testdata/golden/mock-control-f02528f"), buildGolden(t, mockControl, "f02528f")
	homeMoved(t, golden, got)
	compare(t, golden, got, nil)
}

func TestGoldenPhoropter(t *testing.T) {
	golden, got := pagesIn(t, "../../testdata/golden/phoropter-8b23da6"), buildGolden(t, "../..", "8b23da6")
	homeMoved(t, golden, got)
	compare(t, golden, got, nil)
}

// The dev server's pages: a draft with prompts and notes, and a landed plan
// with notes, on a clone of mock-control with the golden drafts/ in it.
func TestGoldenServed(t *testing.T) {
	if _, err := os.Stat(mockControl); err != nil {
		t.Skip("no ../mock-control")
	}
	// Named as the golden pages' clone was, since a site is named for its directory.
	clone := filepath.Join(t.TempDir(), "golden-clone")
	if out, err := exec.Command("git", "clone", "-q", mockControl, clone).CombinedOutput(); err != nil {
		t.Fatal(string(out))
	}
	os.MkdirAll(filepath.Join(clone, "drafts"), 0o755)
	inputs, _ := os.ReadDir("../../testdata/golden/drafts-input")
	for _, e := range inputs {
		b, _ := os.ReadFile(filepath.Join("../../testdata/golden/drafts-input", e.Name()))
		os.WriteFile(filepath.Join(clone, "drafts", e.Name()), b, 0o644)
	}
	repo := &Repo{Dir: clone}
	served, err := Serve(ServeOptions{Repo: repo, Ref: "f02528f", Out: t.TempDir(), Port: 4290, Store: NewStore(repo, t.TempDir(), nil), Guide: testGuide})
	if err != nil {
		t.Fatal(err)
	}
	defer served.Server.Close()
	got := map[string]string{}
	golden := pagesIn(t, "../../testdata/golden/served-mock-control")
	for _, rel := range append(slices.Collect(maps.Keys(golden)), lists...) {
		res, err := http.Get(served.URL + strings.TrimSuffix(rel, "index.html"))
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(res.Body)
		res.Body.Close()
		got[rel] = string(b)
	}
	// When the draft was read is the one value that differs by nature. The
	// lenses link came after the golden pages were made (percale).
	skip := func(l string) bool { return strings.Contains(l, ".readAt[") || strings.HasPrefix(l, "link lenses ->") }
	homeMoved(t, golden, got)
	compare(t, golden, got, skip)
}
