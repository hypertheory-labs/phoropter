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
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
	drive(t, m, m.Init())

	if s := screenText(t, m); !strings.Contains(s, "pumice") || !strings.Contains(s, "tallow") {
		t.Fatalf("plan list:\n%s", s)
	}
	press(t, m, "enter", "j", "j")
	s := screenText(t, m)
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
	if s := screenText(t, m); !strings.Contains(s, "pumice") {
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
	if s := screenText(t, m); !strings.Contains(s, "✎ draft") {
		t.Fatalf("plan list should start with the draft:\n%s", s)
	}
	press(t, m, "enter")
	s := screenText(t, m)
	for _, want := range []string{"draft, not committed, read at 16:12 UTC · anchors at f02528f", "About this draft", "✎ General (Jeff)", "✓ Question: Should the corner", "✎ L14 (Jeff)", "? Select: Which corners count?"} {
		if !strings.Contains(s, want) {
			t.Fatalf("draft screen lacks %q:\n%s", want, s)
		}
	}
	press(t, m, "G")
	if s := screenText(t, m); !strings.Contains(s, "• only the bottom two") || !strings.Contains(s, "Not answered yet") {
		t.Fatalf("select prompt:\n%s", s)
	}
	// r reads the draft again and lands on the entry the prompt follows.
	press(t, m, "r")
	if s := screenText(t, m); !strings.Contains(s, "Decline: A drag handle") || len(m.stack) != 2 {
		t.Fatalf("after refresh (%d screens):\n%s", len(m.stack), s)
	}
}

// Answering writes plan.draft.notes.md beside the draft, by docs/annotating.md,
// and never the draft. The draft source is the file the fixture's page was
// served from, so its sha256 matches the page's header.
func TestAnswerADraft(t *testing.T) {
	src, err := os.ReadFile("../../testdata/mock-control-draft.source.md")
	if err != nil {
		t.Fatal(err)
	}
	repo := t.TempDir()
	os.Mkdir(filepath.Join(repo, ".git"), 0o755)
	os.WriteFile(filepath.Join(repo, "plan.draft.md"), src, 0o644)

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
	// A note on the Decline, then on the whole draft.
	press(t, m, "G", "k", "a", "W", "h", "y", "?", "ctrl+s")
	press(t, m, "home", "a", "G", "o", ".", "ctrl+s")
	notes, err := os.ReadFile(filepath.Join(repo, "plan.draft.notes.md"))
	if err != nil {
		t.Fatal(err, m.status)
	}
	want := []string{
		"# Notes on plan.draft.md\n<!-- annotates: plan.draft.md sha256:",
		"\n\n## — jeff\n\nGo.\n",
		"\n\n## L18 — jeff\n> ## Decline: A drag handle\n\nWhy?\n",
		"\n\n## Select: Which corners count?\n- [x] all four\n- [ ] only the bottom two\n**Note:** quiet\n",
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
	if b, _ := os.ReadFile(filepath.Join(repo, "plan.draft.md")); string(b) != string(src) {
		t.Error("the draft was changed")
	}
}
