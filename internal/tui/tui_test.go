package tui

import (
	"os"
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
	if s := screenText(t, m); !strings.Contains(s, "match-line-prose") {
		t.Fatalf("gaps:\n%s", s)
	}
}
