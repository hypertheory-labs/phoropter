package site

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const testDraft = "---\nbase: f62365f\nrevisits: plans/2026-10-30-gimlet.md\n---\n\n# A draft\n\nWhy now.\n\n## Decision: Serve it\n- where: `src/serve.ts`\n- quote: `const { pages, title } = build(args.repo, args.ref, args.out);`\n\n**Because (Claude, not asked):** a reason\nthat runs two lines.\n\n### Question: Which port?\nSay anything.\n\n**If unanswered:** 4173.\n\n## Decline: Two drafts\n**Because (Jeff):** a backlog.\n\n### Select: Which of these?\n- One\n- Two\n\n**If unanswered:** neither.\n"

const testNotes = "# Notes on plan.draft.md\n<!-- annotates: plan.draft.md sha256:000000000000 -->\n\n## — jeff\n\nThe whole thing reads well.\n\n## L14-15 — jeff\n> **Because (Claude, not asked):** a reason\n> that runs two lines.\n\nWhy two?\n\n## L20 — jeff\n> ## Decline: Two drafts\n> …\n> **Because (Jeff):** a backlog.\n\nAgreed.\n\n## L3 — jeff\n> text that was never there\n\nHm.\n\n## Question: Which port?\n**Answer:** 4174, 4173 is taken.\n"

func TestSplitPrompts(t *testing.T) {
	rest, prompts := SplitPrompts("Before.\n\n### Affirm: Ship it?\n**If unanswered:** yes,\nquietly.\n\n## Next\nAfter.")
	if rest != "Before.\n\n## Next\nAfter." || len(prompts) != 1 || prompts[0].Kind != "Affirm" || prompts[0].Text != "Ship it?" || prompts[0].Fallback != "yes,\nquietly." {
		t.Errorf("rest %q prompts %+v", rest, prompts)
	}
	_, ps := SplitPrompts("#### select: Pick\n- a\n- b\n\nNot an option.")
	if ps[0].Kind != "Select" || !reflect.DeepEqual(ps[0].Options, []string{"a", "b"}) || ps[0].Body != "Not an option." {
		t.Errorf("select = %+v", ps[0])
	}
}

func TestParseNotes(t *testing.T) {
	notes := ParseNotes(testNotes)
	whole, quoted := notes[0], notes[1]
	if whole.Heading != "— jeff" || whole.HasQuote || whole.Comment != "The whole thing reads well." {
		t.Errorf("whole = %+v", whole)
	}
	if quoted.Quote != "**Because (Claude, not asked):** a reason\nthat runs two lines." || quoted.Comment != "Why two?" {
		t.Errorf("quoted = %+v", quoted)
	}
}

// A draft resolves against the site without joining it: its anchors resolve
// at its base and now, its prompts get their answers from restated headings,
// and its notes are placed after the entries they quote.
func TestReadDraft(t *testing.T) {
	repo := &Repo{Dir: "../.."}
	s, err := BuildSite(repo, "f62365f", NewStore(repo, t.TempDir(), nil))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "drafts"), 0o755)
	os.WriteFile(filepath.Join(dir, DraftFile), []byte(testDraft), 0o644)
	d, _ := ReadDraft(s, dir)
	if d.Notes != nil {
		t.Fatal("no notes yet")
	}
	os.WriteFile(filepath.Join(dir, DraftNotes), []byte(testNotes), 0o644)
	d, _ = ReadDraft(s, dir)
	if d.Plan.Revisits == nil || d.Plan.Revisits.Name != "gimlet" || len(d.Plan.Revisits.RevisitedBy) != 0 {
		t.Errorf("revisits = %+v", d.Plan.Revisits)
	}
	serve := d.Plan.Doc.Entries[0]
	if v := serve.Resolved[0].Views; len(v) != 1 || v[0].Role != "generated" || v[0].Status != "found" || strings.Contains(serve.Body, "Question") {
		t.Errorf("serve = %+v %+v", serve, v)
	}
	q := d.Prompts["serve-it"][0]
	if q.Fallback != "4173." || q.Answer == nil || q.Answer.Comment != "**Answer:** 4174, 4173 is taken." {
		t.Errorf("question = %+v", q)
	}
	if !reflect.DeepEqual(d.Prompts["two-drafts"][0].Options, []string{"One", "Two"}) {
		t.Errorf("select = %+v", d.Prompts["two-drafts"][0])
	}
	var at []string
	for _, a := range d.Notes.Annotations {
		at = append(at, a.Heading+"|"+map[bool]string{true: "found", false: "lost"}[a.Found]+"|"+a.At)
	}
	if strings.Join(at, ";") != "— jeff|found|;L14-15 — jeff|found|serve-it;L20 — jeff|found|two-drafts;L3 — jeff|lost|" {
		t.Errorf("notes = %v", at)
	}
	if d.Notes.Annotates != "000000000000" || d.Notes.Current {
		t.Errorf("annotates = %+v", d.Notes)
	}
	// The hash annotations.nvim writes: the lines joined, without the final newline.
	nvim := sha(strings.TrimSuffix(testDraft, "\n"))[:12]
	os.WriteFile(filepath.Join(dir, DraftNotes), []byte(strings.Replace(testNotes, "000000000000", nvim, 1)), 0o644)
	if d, _ = ReadDraft(s, dir); !d.Notes.Current {
		t.Error("notes should be current")
	}
	h := DraftPage(s, d).HTML
	for _, want := range []string{`Draft, not committed: <code itemprop="file">drafts/plan.draft.md</code>`, "quote not found: it moved under the note"} {
		if !strings.Contains(h, want) {
			t.Errorf("draft page lacks %q", want)
		}
	}
	if strings.Index(h, "Why two?") > strings.Index(h, "Two drafts</span>") || strings.Index(h, "Agreed.") < strings.Index(h, "Two drafts</span>") {
		t.Error("notes aren't after the entries they quote")
	}
}

// tarlatan: a landed plan's notes are kept in drafts/, placed after the
// entries they quote, using the plan's real file.
func TestReadPlanNotes(t *testing.T) {
	repo := &Repo{Dir: "../.."}
	s, err := BuildSite(repo, "f62365f", NewStore(repo, t.TempDir(), nil))
	if err != nil {
		t.Fatal(err)
	}
	var gimlet *Plan
	for _, p := range s.Plans {
		if p.Name == "gimlet" {
			gimlet = p
		}
	}
	if PlanNotesFile(gimlet) != "drafts/2026-10-30-gimlet.notes.md" {
		t.Errorf("file = %s", PlanNotesFile(gimlet))
	}
	dir := t.TempDir()
	if _, ok := ReadPlanNotes(dir, gimlet); ok {
		t.Error("no notes yet")
	}
	os.MkdirAll(filepath.Join(dir, "drafts"), 0o755)
	os.MkdirAll(filepath.Join(dir, "plans"), 0o755)
	real, _ := repo.Show(s.Generated.SHA, gimlet.File)
	os.WriteFile(filepath.Join(dir, gimlet.File), []byte(real), 0o644)
	os.WriteFile(filepath.Join(dir, PlanNotesFile(gimlet)), []byte("# Notes\n\n## — jeff\n\nStill true?\n\n## L3 — jeff\n> ## Decline: A separate client app\n\nThe TUI moved in.\n"), 0o644)
	notes, _ := ReadPlanNotes(dir, gimlet)
	var slug string
	for _, e := range gimlet.Doc.Entries {
		if e.Title == "A separate client app" {
			slug = e.Slug
		}
	}
	if len(notes) != 2 || notes[0].At != "" || notes[1].At != slug {
		t.Errorf("notes = %+v (want %s)", notes, slug)
	}
}
