package site

import (
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// poplin: served at HEAD, the site follows HEAD; a request after a commit
// builds the new one first.
func TestServerFollowsHEAD(t *testing.T) {
	dir := t.TempDir()
	git := func(args ...string) string {
		out, err := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@t"}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatal(string(out))
		}
		return strings.TrimSpace(string(out))
	}
	commit := func(name string) string {
		os.WriteFile(filepath.Join(dir, name), []byte(name+"\n"), 0o644)
		git("add", name)
		git("commit", "-q", "-m", name)
		return git("rev-parse", "HEAD")
	}
	git("init", "-q")
	first := commit("a.txt")
	repo := &Repo{Dir: dir}
	rebuilt := 0
	served, err := Serve(ServeOptions{Repo: repo, Ref: "HEAD", Out: filepath.Join(t.TempDir(), "site"), Port: 4310, Store: NewStore(repo, t.TempDir(), nil), Guide: testGuide,
		OnRebuild: func(*Site, int) { rebuilt++ }})
	if err != nil {
		t.Fatal(err)
	}
	defer served.Server.Close()
	commitOf := regexp.MustCompile(`itemprop="commit" value="([0-9a-f]+)"`)
	at := func() string {
		res, err := http.Get(served.URL)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(res.Body)
		res.Body.Close()
		m := commitOf.FindStringSubmatch(string(b))
		if m == nil {
			t.Fatalf("no commit on the home page:\n%s", b)
		}
		return m[1]
	}
	if got := at(); got != first {
		t.Fatalf("at %s, want %s", got, first)
	}
	second := commit("b.txt")
	time.Sleep(1100 * time.Millisecond)
	if got := at(); got != second || rebuilt != 1 {
		t.Errorf("after a commit: at %s (want %s), %d rebuilds", got, second, rebuilt)
	}
	if got := at(); got != second || rebuilt != 1 {
		t.Errorf("rebuilt again with nothing new: %d", rebuilt)
	}
	if err := served.Rebuild(); err != nil || rebuilt != 2 {
		t.Errorf("Rebuild: %v, %d rebuilds", err, rebuilt)
	}
	if old, _ := filepath.Glob(filepath.Join(filepath.Dir(served.live.root), "site*")); len(old) != 1 {
		t.Errorf("old sites left behind: %v", old)
	}
}

// poplin, from shelf/draft-places.md: each draft anchor lists the landed
// entries at its place, open guesses first, then declines, then decisions.
func TestDraftShowsWhatsAlreadyHere(t *testing.T) {
	repo := &Repo{Dir: "../.."}
	s, err := BuildSite(repo, "51b503c", NewStore(repo, t.TempDir(), nil))
	if err != nil {
		t.Fatal(err)
	}
	var place *Place
	for _, p := range s.Places {
		if p.Kind == "file" && (place == nil || len(p.Refs) > len(place.Refs)) {
			place = p
		}
	}
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "drafts"), 0o755)
	os.WriteFile(filepath.Join(dir, DraftFile), []byte("---\nbase: 51b503c\n---\n\n# A draft\n\n## Decision: Something here\n- where: `"+place.Full+"`\n\n**Because (t):** a test.\n"), 0o644)
	d, err := ReadDraft(s, dir)
	if err != nil {
		t.Fatal(err)
	}
	page := DraftPage(s, d).HTML
	links := regexp.MustCompile(`itemprop="alreadyHere" href="[^"]*">(\w+):`).FindAllStringSubmatch(page, -1)
	seen := map[*ResolvedEntry]bool{}
	for _, r := range place.Refs {
		seen[r.Entry] = true
	}
	if len(links) == 0 || len(links) != len(seen) {
		t.Fatalf("%d links for %d entries at %s", len(links), len(seen), place.Full)
	}
	order := map[string]int{"Decline": 1, "Decision": 2}
	last := 0
	for _, l := range links {
		o, ok := order[l[1]]
		if !ok {
			o = 3
		}
		if l[1] == "Guess" {
			o = -1 // open or not, only checked to come before what ranks below it
		}
		if o >= 0 && o < last {
			t.Errorf("%s after rank %d: %v", l[1], last, links)
		}
		if o > last {
			last = o
		}
	}
}

// An empty repository's home page says to run init; once init has written
// the draft, the dev server points at the draft instead (sateen).
func TestWelcomeSaysInitThenTheDraft(t *testing.T) {
	dir := t.TempDir()
	page := `<a rel="home" href="./">home</a>` + "\n" + welcomeStart
	h := &handler{repoDir: dir}
	if got := h.withDraftLink(page, ""); !strings.Contains(got, "phoropter init") {
		t.Errorf("no draft, but the welcome lost init:\n%s", got)
	}
	os.MkdirAll(filepath.Join(dir, "drafts"), 0o755)
	os.WriteFile(filepath.Join(dir, DraftFile), []byte("# x\n"), 0o644)
	got := h.withDraftLink(page, "")
	if strings.Contains(got, "phoropter init") || !strings.Contains(got, `<a href="draft/">read it at`) {
		t.Errorf("with a draft, the welcome should point at it:\n%s", got)
	}
	if n := strings.Count(got, `rel="draft"`); n > 1 {
		t.Errorf("%d rel=draft links; the welcome's link shouldn't add one", n)
	}
}

// A rebuild that fails leaves the last site up, and every page says so until
// one succeeds; HEAD coming back to the served commit clears it (taffeta).
func TestAPageSaysHEADCouldntBeBuilt(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root can write a read-only directory")
	}
	dir := t.TempDir()
	git := func(args ...string) string {
		out, err := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@t"}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatal(string(out))
		}
		return strings.TrimSpace(string(out))
	}
	commit := func(name string) string {
		os.WriteFile(filepath.Join(dir, name), []byte(name+"\n"), 0o644)
		git("add", name)
		git("commit", "-q", "-m", name)
		return git("rev-parse", "HEAD")
	}
	git("init", "-q")
	first := commit("a.txt")
	repo := &Repo{Dir: dir}
	// A rebuild writes beside the site, so a read-only parent makes it fail.
	parent := t.TempDir()
	served, err := Serve(ServeOptions{Repo: repo, Ref: "HEAD", Out: filepath.Join(parent, "site"), Port: 4320, Store: NewStore(repo, t.TempDir(), nil), Guide: testGuide})
	if err != nil {
		t.Fatal(err)
	}
	defer served.Server.Close()
	defer os.Chmod(parent, 0o755)
	get := func(p string) string {
		res, err := http.Get(served.URL + p)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(res.Body)
		res.Body.Close()
		return string(b)
	}
	if strings.Contains(get(""), `itemprop="stale"`) {
		t.Fatal("stale before anything failed")
	}

	os.Chmod(parent, 0o555)
	second := commit("b.txt")
	time.Sleep(1100 * time.Millisecond)
	for _, p := range []string{"", "guide/", "lenses/"} {
		page := get(p)
		if !strings.Contains(page, `itemprop="stale"`) || !strings.Contains(page, `value="`+second+`"`) || !strings.Contains(page, `itemprop="commit" value="`+first+`"`) {
			t.Errorf("%q doesn't say HEAD %s failed while serving %s:\n%.1500s", p, second[:7], first[:7], page)
		}
	}

	git("reset", "-q", "--hard", first)
	time.Sleep(1100 * time.Millisecond)
	if strings.Contains(get(""), `itemprop="stale"`) {
		t.Error("still stale with HEAD back at the served commit")
	}

	os.Chmod(parent, 0o755)
	third := commit("c.txt")
	time.Sleep(1100 * time.Millisecond)
	if page := get(""); strings.Contains(page, `itemprop="stale"`) || !strings.Contains(page, `value="`+third+`"`) {
		t.Errorf("after a commit that builds:\n%.1500s", page)
	}
}
