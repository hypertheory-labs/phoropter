package notes

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const draft = `---
base: f02528f
---

# Let the panel remember which corner it was in

## Decision: The panel stores its corner in localStorage
- where: ` + "`src/panel.ts`" + `

**Because (Claude, not asked):** people move it out of the way and it comes back.

### Question: Should the corner be per app or ` + "`global`" + `?
**If unanswered:** per app, keyed by origin.

## Decline: A drag handle
**Because (Claude, not asked):** four corners are enough.

### Select: Which corners count?
- all four
- only the bottom two

**If unanswered:** all four.
`

// The shape annotations.nvim writes: a title, its own comments, and notes by
// line range with quotes.
const nvimNotes = `# Notes on plan.draft.md
<!-- annotates: plan.draft.md sha256:000000000000 -->
<!-- Protocol: do not edit plan.draft.md while this file exists. -->

## L10

> **Because (Claude, not asked):** people move it out of the way and it comes back.

Is localStorage everywhere?

## L16-17

> ## Decline: A drag handle

Fine by me.
`

func TestSourceHeadings(t *testing.T) {
	s := ParseSource(draft)
	if len(s.Prompts) != 2 || len(s.Entries) != 2 {
		t.Fatalf("prompts %+v entries %+v", s.Prompts, s.Entries)
	}
	// The page renders the backticks away; the source keeps them.
	p, ok := s.Prompt("question", "Should the corner be per app or global?")
	if !ok || p.Text != "Should the corner be per app or `global`?" || p.Line != 12 {
		t.Errorf("prompt = %+v %v", p, ok)
	}
	if s.Entries[1].Raw != "## Decline: A drag handle" || s.Entries[1].Line != 15 {
		t.Errorf("entry = %+v", s.Entries[1])
	}
}

func TestRoundTripKeepsWhatItDidntWrite(t *testing.T) {
	if got := Parse(nvimNotes).String(); got != nvimNotes {
		t.Errorf("round trip changed the file:\n%s", got)
	}
}

func TestAnswerGoesInDocumentOrder(t *testing.T) {
	src := ParseSource(draft)
	f := Parse(nvimNotes)
	q, _ := src.Prompt("Question", "Should the corner be per app or global?")
	f.Answer(src, q, []string{"**Answer:** Per app."})
	sel, _ := src.Prompt("Select", "Which corners count?")
	f.Answer(src, sel, []string{"- [x] all four", "- [ ] only the bottom two", "**Note:** top two hold the browser's UI."})
	f.Comment(src, nil, "jeff", "Small and good.")
	var heads []string
	for _, n := range f.Notes {
		heads = append(heads, n.Heading)
	}
	want := "— jeff | L10 | Question: Should the corner be per app or `global`? | L16-17 | Select: Which corners count?"
	if got := strings.Join(heads, " | "); got != want {
		t.Errorf("order:\n got %s\nwant %s", got, want)
	}
	// Answering again replaces, and moves nothing else.
	f.Answer(src, q, []string{"**Skipped.**"})
	if len(f.Notes) != 5 || !strings.Contains(f.String(), "## Question: Should the corner be per app or `global`?\n**Skipped.**\n") || strings.Contains(f.String(), "Per app.") {
		t.Errorf("re-answer:\n%s", f.String())
	}
}

func TestCommentOnAnEntryQuotesItsHeading(t *testing.T) {
	src := ParseSource(draft)
	f := &File{}
	f.Comment(src, &src.Entries[1], "jeff", "Why not?")
	f.SetAnnotates("plan.draft.md", draft)
	sum := sha256.Sum256([]byte(strings.TrimSuffix(draft, "\n")))
	want := "# Notes on plan.draft.md\n<!-- annotates: plan.draft.md sha256:" + hex.EncodeToString(sum[:])[:12] + " -->\n\n## L15 — jeff\n> ## Decline: A drag handle\n\nWhy not?\n"
	if got := f.String(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestOpenChecksThePagesHash(t *testing.T) {
	root := t.TempDir()
	os.Mkdir(filepath.Join(root, ".git"), 0o755)
	os.WriteFile(filepath.Join(root, "plan.draft.md"), []byte(draft), 0o644)
	sub := filepath.Join(root, "internal")
	os.Mkdir(sub, 0o755)
	sum := sha256.Sum256([]byte(draft))
	if _, err := Open(sub, "plan.draft.md", "beef"); err != ErrMoved {
		t.Errorf("a different hash should be ErrMoved, got %v", err)
	}
	d, err := Open(sub, "plan.draft.md", hex.EncodeToString(sum[:]))
	if err != nil || d.HasNotes {
		t.Fatalf("open = %+v, %v", d, err)
	}
	d.Notes.Comment(d.Source, nil, "jeff", "Go.")
	if err := d.Save(); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(root, "plan.draft.notes.md")); !strings.Contains(string(b), "## — jeff\n\nGo.\n") {
		t.Errorf("notes file:\n%s", b)
	}
	if b, _ := os.ReadFile(filepath.Join(root, "plan.draft.md")); string(b) != draft {
		t.Error("the draft was changed")
	}
}
