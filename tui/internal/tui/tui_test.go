package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"phoropter-tui/internal/substrate"
)

// drive runs a command to completion, the way the program loop would.
func drive(t *testing.T, m *Model, cmd tea.Cmd) {
	t.Helper()
	for cmd != nil {
		msg := cmd()
		if e, ok := msg.(errMsg); ok {
			t.Fatal(e.err)
		}
		_, cmd = m.Update(msg)
	}
}

func press(t *testing.T, m *Model, keys ...string) {
	t.Helper()
	for _, k := range keys {
		var msg tea.KeyMsg
		switch k {
		case "enter":
			msg = tea.KeyMsg{Type: tea.KeyEnter}
		case "esc":
			msg = tea.KeyMsg{Type: tea.KeyEsc}
		case "tab":
			msg = tea.KeyMsg{Type: tea.KeyTab}
		case "ctrl+s":
			msg = tea.KeyMsg{Type: tea.KeyCtrlS}
		case " ":
			msg = tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}
		default:
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
		}
		_, cmd := m.Update(msg)
		drive(t, m, cmd)
	}
}

func screenText(t *testing.T, m *Model) string {
	t.Helper()
	s := ansi.Strip(m.View())
	if os.Getenv("SHOW") != "" {
		t.Log("\n" + s)
	}
	return s
}

func TestWalkAPlan(t *testing.T) {
	c, err := substrate.NewClient("../../testdata/mock-control-f02528f", substrate.NewLedger())
	if err != nil {
		t.Fatal(err)
	}
	m := New(c, c.Ledger)
	m.dir = t.TempDir() // no draft on disk, whatever this checkout has
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
	drive(t, m, m.Init())

	// Titles first, newest first, the empty draft slot on top.
	s := screenText(t, m)
	if !strings.Contains(s, "No draft") || !strings.Contains(s, "Plans · newest first") || strings.Index(s, "Revisit pumice: MSW 3") > strings.Index(s, "Get mock-control ready to publish") || !strings.Contains(s, "pumice · 2026-10-29 · 9 entries · ✓ as-built") {
		t.Fatalf("plan list:\n%s", s)
	}
	// o flips the order and keeps tallow chosen; home is the slot, j is pumice.
	press(t, m, "o")
	if s := screenText(t, m); !strings.Contains(s, "oldest first") || !strings.Contains(s, "revisits    pumice") {
		t.Fatalf("after o:\n%s", s)
	}
	press(t, m, "home", "j", "enter", "j", "j")
	s = screenText(t, m)
	if !strings.Contains(s, "Get mock-control ready to publish") || !strings.Contains(s, "1 before") {
		t.Fatalf("pumice by its title, layers by their names:\n%s", s)
	}
	s = screenText(t, m)
	if !strings.Contains(s, "selector: 'app-mock-panel'") {
		t.Fatalf("base layer should show the old selector:\n%s", s)
	}
	press(t, m, "2")
	if s := screenText(t, m); !strings.Contains(s, "selector: 'mock-panel'") {
		t.Fatalf("as-built head should show the rename:\n%s", s)
	}
	press(t, m, "3")
	if s := screenText(t, m); !strings.Contains(s, "code repeated from e92db76") {
		t.Fatalf("generated should be the head's code:\n%s", s)
	}

	// Follow the anchor to the symbol's page, then back into a plan.
	press(t, m, "enter")
	screenText(t, m)
	press(t, m, "enter")
	if s := screenText(t, m); !strings.Contains(s, "MockPanel#") {
		t.Fatalf("place:\n%s", s)
	}
	press(t, m, "enter")
	if s := screenText(t, m); !strings.Contains(s, "Get mock-control ready") {
		t.Fatalf("back into a plan:\n%s", s)
	}

	// A guess with a (new) anchor explains its empty layer.
	press(t, m, "esc", "esc", "G")
	press(t, m, "k", "k")
	screenText(t, m)
	press(t, m, "esc", "j", "j", "j", "enter")
	screenText(t, m)
	press(t, m, "g")
	if s := screenText(t, m); !strings.Contains(s, "new-never-resolved") {
		t.Fatalf("gaps:\n%s", s)
	}
}

func TestReadADraft(t *testing.T) {
	c, err := substrate.NewClient("../../testdata/mock-control-draft", substrate.NewLedger())
	if err != nil {
		t.Fatal(err)
	}
	m := New(c, c.Ledger)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
	drive(t, m, m.Init())
	if s := screenText(t, m); !strings.Contains(s, "✎ Let the panel remember") || !strings.Contains(s, "draft, read at 18:24 UTC") {
		t.Fatalf("the slot should hold the draft:\n%s", s)
	}
	press(t, m, "enter")
	s := screenText(t, m)
	for _, want := range []string{"draft: Let the panel remember which corner it was in · not committed, read at 18:24 UTC", "About this draft · a to note it", "✎ General (Jeff)", "✓ Question: Should the corner", "✎ L14 (Jeff)", "✓ Select: Which corners count?"} {
		if !strings.Contains(s, want) {
			t.Fatalf("draft screen lacks %q:\n%s", want, s)
		}
	}
	press(t, m, "G")
	if s := screenText(t, m); !strings.Contains(s, "• only the bottom two") || !strings.Contains(s, "[x] all four") || !strings.Contains(s, "[ ] only the bottom two") {
		t.Fatalf("select prompt:\n%s", s)
	}
	// r reads the draft again and lands on the entry the prompt follows.
	press(t, m, "r")
	if s := screenText(t, m); !strings.Contains(s, "Decline: A drag handle") || len(m.stack) != 2 {
		t.Fatalf("after refresh (%d screens):\n%s", len(m.stack), s)
	}
}

// Answering writes drafts/plan.draft.notes.md beside the draft, by docs/annotating.md,
// and never the draft. The draft source is the file the fixture's page was
// served from, so its sha256 matches the page's header.
func TestAnswerADraft(t *testing.T) {
	src, err := os.ReadFile("../../testdata/mock-control-draft.source.md")
	if err != nil {
		t.Fatal(err)
	}
	repo := t.TempDir()
	os.Mkdir(filepath.Join(repo, ".git"), 0o755)
	os.Mkdir(filepath.Join(repo, "drafts"), 0o755)
	os.WriteFile(filepath.Join(repo, "drafts", "plan.draft.md"), src, 0o644)

	c, err := substrate.NewClient("../../testdata/mock-control-draft", substrate.NewLedger())
	if err != nil {
		t.Fatal(err)
	}
	m := New(c, c.Ledger)
	m.dir, m.authorName = repo, "jeff"
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
	drive(t, m, m.Init())
	press(t, m, "enter", "G", "a")
	s := screenText(t, m)
	if !strings.Contains(s, "Answer · Select: Which corners count?") || !strings.Contains(s, "fixes the draft") {
		t.Fatalf("select form:\n%s", s)
	}
	// q is typed into the form, not quit.
	press(t, m, " ", "j", "tab", "q", "u", "i", "e", "t", "ctrl+s")
	if len(m.stack) != 2 {
		t.Fatalf("form didn't write: %s\n%s", m.status, screenText(t, m))
	}
	// esc with something typed asks first; a second esc discards.
	press(t, m, "G", "a", " ", "esc")
	if s := screenText(t, m); !strings.Contains(s, "esc again throws away") {
		t.Fatalf("esc should ask first:\n%s", s)
	}
	press(t, m, "esc")
	if s := screenText(t, m); strings.Contains(s, "Answer · Select") {
		t.Fatalf("second esc should close the form:\n%s", s)
	}
	// A note on the Decline, then on the whole draft.
	press(t, m, "G", "k", "a", "W", "h", "y", "?", "ctrl+s")
	press(t, m, "home", "a", "G", "o", ".", "ctrl+s")
	notes, err := os.ReadFile(filepath.Join(repo, "drafts", "plan.draft.notes.md"))
	if err != nil {
		t.Fatal(err, m.status)
	}
	want := []string{
		"# Notes on plan.draft.md\n<!-- annotates: plan.draft.md sha256:",
		"\n\n## — jeff\n\nGo.\n",
		"\n\n## L18 — jeff\n> ## Decline: A drag handle\n\nWhy?\n",
		"\n\n## Select: Which corners count?\n- [x] all four\n- [ ] only the bottom two\n\n**Note:** quiet\n",
	}
	got := string(notes)
	last := -1
	for _, w := range want {
		i := strings.Index(got, w)
		if i < 0 || i < last {
			t.Fatalf("notes lack %q in order:\n%s", w, got)
		}
		last = i
	}
	if b, _ := os.ReadFile(filepath.Join(repo, "drafts", "plan.draft.md")); string(b) != string(src) {
		t.Error("the draft was changed")
	}
}

// What's typed past the pane's edge wraps, so it can be seen (challis).
func TestTextBoxWraps(t *testing.T) {
	f := newNoteForm(-1, "the whole draft", false)
	f.text = []rune(strings.Repeat("word ", 30) + "last")
	s := ansi.Strip(f.view(40))
	if !strings.Contains(s, "last▏") || strings.Contains(s, "…") {
		t.Fatalf("text box:\n%s", s)
	}
}

// tab moves through a list like j, and wraps at the end; shift+tab goes back.
func TestTabCycles(t *testing.T) {
	if got := move(3, 4, "tab"); got != 0 {
		t.Errorf("tab at the end = %d, want 0", got)
	}
	if got := move(0, 4, "shift+tab"); got != 3 {
		t.Errorf("shift+tab at the top = %d, want 3", got)
	}
	if got := move(3, 4, "j"); got != 3 {
		t.Errorf("j at the end = %d, want 3", got)
	}
}

// tarlatan: a landed plan takes notes, never edits. a on an entry writes a
// request to revisit into drafts/, quoting the entry's heading; the plan's
// file is never touched. The request already on pumice shows as rows.
func TestAskForARevisit(t *testing.T) {
	src, err := os.ReadFile("../../testdata/mock-control-pumice.plan.md")
	if err != nil {
		t.Fatal(err)
	}
	repo := t.TempDir()
	os.Mkdir(filepath.Join(repo, ".git"), 0o755)
	os.Mkdir(filepath.Join(repo, "plans"), 0o755)
	plan := filepath.Join(repo, "plans", "2026-10-29-pumice.md")
	os.WriteFile(plan, src, 0o644)

	c, err := substrate.NewClient("../../testdata/mock-control-draft", substrate.NewLedger())
	if err != nil {
		t.Fatal(err)
	}
	m := New(c, c.Ledger)
	m.dir, m.authorName = repo, "jeff"
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
	drive(t, m, m.Init())
	if s := screenText(t, m); !strings.Contains(s, "✎ 2 notes") {
		t.Fatalf("the list should mark pumice's request:\n%s", s)
	}
	m.c.Forget("plans/pumice/")
	drive(t, m, m.open("plans/pumice/"))
	s := screenText(t, m)
	for _, want := range []string{"About this plan · a to ask for a revisit", "✎ — jeff", "✎ L22 — jeff"} {
		if !strings.Contains(s, want) {
			t.Fatalf("pumice lacks %q:\n%s", want, s)
		}
	}
	// The request's whole-plan note is row 1; the entries follow.
	press(t, m, "home", "j", "j", "j", "a")
	if s := screenText(t, m); !strings.Contains(s, "Request to revisit, on · Decision: Selectors") || strings.Contains(s, "fixes the draft") {
		t.Fatalf("request form:\n%s", s)
	}
	press(t, m, "S", "t", "i", "l", "l", "?", "ctrl+s")
	got, err := os.ReadFile(filepath.Join(repo, "drafts", "2026-10-29-pumice.notes.md"))
	if err != nil {
		t.Fatal(err, m.status)
	}
	for _, want := range []string{
		"# Notes on plans/2026-10-29-pumice.md\n<!-- annotates: plans/2026-10-29-pumice.md sha256:",
		"\n\n## L22 — jeff\n> ## Decision: Selectors use the library's prefix, `mock-panel`\n\nStill?\n",
	} {
		if !strings.Contains(string(got), want) {
			t.Fatalf("request lacks %q:\n%s", want, got)
		}
	}
	if b, _ := os.ReadFile(plan); string(b) != string(src) {
		t.Error("the landed plan was changed")
	}
}
