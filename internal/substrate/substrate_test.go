package substrate

import (
	"strings"
	"testing"
)

const mockControl = "../../testdata/mock-control-f02528f"

func client(t *testing.T) (*Client, *Ledger) {
	t.Helper()
	l := NewLedger()
	c, err := NewClient(mockControl, l)
	if err != nil {
		t.Fatal(err)
	}
	return c, l
}

func TestHome(t *testing.T) {
	c, _ := client(t)
	h, err := c.Home()
	if err != nil {
		t.Fatal(err)
	}
	if h.Repository != "mock-control" || h.Commit.Label != "f02528f" {
		t.Errorf("provenance = %q at %q", h.Repository, h.Commit.Label)
	}
	var names []string
	for _, p := range h.Plans {
		names = append(names, p.Name)
	}
	if got := strings.Join(names, " "); got != "pumice vellum ferrule tallow" {
		t.Errorf("plans = %s", got)
	}
	tallow := h.Plans[3]
	if tallow.AsBuiltURL == "" || !strings.HasSuffix(tallow.RevisitsURL, "plans/pumice/") || tallow.Entries != 6 {
		t.Errorf("tallow = %+v", tallow)
	}
	if len(h.Places) != 5 {
		t.Errorf("places = %d", len(h.Places))
	}
}

func TestPlanViews(t *testing.T) {
	c, _ := client(t)
	d, err := c.Doc("plans/pumice/")
	if err != nil {
		t.Fatal(err)
	}
	if d.IsAsBuilt || d.Base.Label != "da023f1" || len(d.Entries) != 9 {
		t.Fatalf("pumice = %+v", d)
	}
	e := d.Entries[1]
	if e.Kind != "Decision" || e.Title != "Selectors use the library's prefix, `mock-panel`" || e.Source != "Jeff" {
		t.Errorf("entry = %q %q %q", e.Kind, e.Title, e.Source)
	}
	views := e.Anchors[0].Views
	if len(views) != 3 {
		t.Fatalf("views = %d", len(views))
	}
	base, head, gen := views[0], views[1], views[2]
	if base.Role != "base" || base.MatchLine != 10 || base.Code.Lines[1].Text != "  selector: 'app-mock-panel'," || !base.Code.Lines[1].Marked {
		t.Errorf("base = %+v %+v", base, base.Code.Lines[:2])
	}
	if head.Status != "not-found" || head.StartLine != 9 || head.Code.Lines[1].Text != "  selector: 'mock-panel'," {
		t.Errorf("head = %+v", head)
	}
	if gen.SameAs != "e92db76" || gen.Code != head.Code {
		t.Errorf("generated should reuse the head's code: %+v", gen)
	}
	guess := d.Entries[7]
	if guess.Kind != "Guess" || !guess.Anchors[0].New || len(guess.AnchoredBy) != 1 {
		t.Errorf("guess = %+v", guess)
	}
}

func TestAsBuiltVerdict(t *testing.T) {
	c, _ := client(t)
	d, err := c.Doc("plans/vellum/as-built/")
	if err != nil {
		t.Fatal(err)
	}
	if !d.IsAsBuilt || !strings.HasSuffix(d.PlanURL, "plans/vellum/") {
		t.Fatalf("as-built = %+v", d)
	}
	var verdicts []string
	for _, e := range d.Entries {
		if e.Kind == "Answered" {
			verdicts = append(verdicts, e.Verdict)
			if e.Answers == nil {
				t.Errorf("%s has no answers link", e.Title)
			}
		}
	}
	if got := strings.Join(verdicts, ","); got != "held,disproved" {
		t.Errorf("verdicts = %s", got)
	}
}

func TestSymbolPlace(t *testing.T) {
	c, _ := client(t)
	p, err := c.Place("symbols/projects/mock-control/src/lib/panel/mock-panel.ts/MockPanel/")
	if err != nil {
		t.Fatal(err)
	}
	if p.Kind != "symbol" || p.StartLine != 9 || p.Code.Lines[0].N != 9 || len(p.AnchoredBy) == 0 {
		t.Errorf("place = %+v", p)
	}
	if !strings.HasSuffix(p.FileURL, "files/projects/mock-control/src/lib/panel/mock-panel.ts/") {
		t.Errorf("file url = %s", p.FileURL)
	}
}

func TestCrawlLedger(t *testing.T) {
	c, l := client(t)
	if _, err := c.Crawl(); err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, g := range l.Gaps() {
		seen[g.ID] = true
	}
	for _, id := range []string{"match-line-prose", "home-as-built-unmarked", "home-revisits-unmarked", "new-never-resolved", "symbol-file-unlinked", "answer-elsewhere", "same-as-deref"} {
		if !seen[id] {
			t.Errorf("crawl didn't note %s", id)
		}
	}
}

func TestPhoropterCrawls(t *testing.T) {
	c, err := NewClient("../../testdata/phoropter-e8107f3", NewLedger())
	if err != nil {
		t.Fatal(err)
	}
	s, err := c.Crawl()
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Home.Plans) != 7 || len(s.Docs) != 14 {
		t.Errorf("plans = %d, docs = %d", len(s.Home.Plans), len(s.Docs))
	}
	if s.Home.Plans[0].Base.SHA != "" {
		t.Errorf("gimlet's base should be text: %+v", s.Home.Plans[0].Base)
	}
}
