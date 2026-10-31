package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"phoropter-tui/internal/notes"
	"phoropter-tui/internal/substrate"
)

// A form answers a draft's prompt or writes a note, into the notes file beside
// the draft. It's the TUI's only write, and it never writes the draft.
type form struct {
	prompt  *substrate.Prompt // nil for a note
	entry   int               // a note's entry, by index; -1 for the whole draft
	about   string            // what's being answered or noted, for the heading
	checked []bool            // a Select's
	optCur  int
	yes     *bool  // an Affirm's
	text    []rune // a Question's answer, or a note's comment
	note    []rune // **Note:** on any answer
	field   int    // 0 the answer, 1 the note
	first   bool   // no notes file yet, so writing fixes the draft
	request bool   // a note on a landed plan, asking for a revisit
	leaving bool   // esc once with something typed; esc again discards
	err     string
}

func newPromptForm(p *substrate.Prompt, first bool) *form {
	return &form{prompt: p, about: p.Kind + ": " + p.Text, checked: make([]bool, len(p.Options)), first: first}
}

func newNoteForm(entry int, about string, first bool) *form {
	return &form{entry: entry, about: about, first: first}
}

func (f *form) fields() int {
	if f.prompt == nil {
		return 1
	}
	return 2
}

func (f *form) textField() *[]rune {
	if f.field == 1 {
		return &f.note
	}
	if f.prompt == nil || f.prompt.Kind == "Question" {
		return &f.text
	}
	return nil
}

func (f *form) help() string {
	return "ctrl+s write · tab next field · ctrl+e $EDITOR · esc cancel"
}

// key handles everything typed while the form is open. It reports whether
// the form asked to write or to cancel.
func (f *form) key(msg tea.KeyMsg) (write, cancel bool, edit *[]rune) {
	f.err = ""
	leaving := f.leaving
	f.leaving = false
	switch msg.String() {
	case "esc":
		if f.dirty() && !leaving {
			f.leaving = true
			f.err = "esc again throws away what's here; any other key keeps it"
			return
		}
		return false, true, nil
	case "ctrl+s":
		return true, false, nil
	case "tab", "shift+tab":
		f.field = (f.field + 1) % f.fields()
		return
	case "ctrl+e":
		if t := f.textField(); t != nil {
			return false, false, t
		}
		return
	}
	if t := f.textField(); t != nil {
		switch msg.Type {
		case tea.KeyRunes, tea.KeySpace:
			*t = append(*t, msg.Runes...)
		case tea.KeyEnter:
			*t = append(*t, '\n')
		case tea.KeyBackspace:
			if len(*t) > 0 {
				*t = (*t)[:len(*t)-1]
			}
		}
		return
	}
	switch f.prompt.Kind {
	case "Affirm":
		switch msg.String() {
		case "y":
			v := true
			f.yes = &v
		case "n":
			v := false
			f.yes = &v
		}
	case "Select":
		switch msg.String() {
		case " ", "x":
			f.checked[f.optCur] = !f.checked[f.optCur]
		default:
			f.optCur = move(f.optCur, len(f.checked), msg.String())
		}
	}
	return
}

// dirty is whether closing the form would lose anything.
func (f *form) dirty() bool {
	if len(f.text) > 0 || len(f.note) > 0 || f.yes != nil {
		return true
	}
	for _, c := range f.checked {
		if c {
			return true
		}
	}
	return false
}

// lines are what goes under the restated heading, by docs/annotating.md.
func (f *form) lines() ([]string, error) {
	var out []string
	switch f.prompt.Kind {
	case "Question":
		answer := strings.TrimSpace(string(f.text))
		if answer == "" {
			return nil, fmt.Errorf("write an answer, or esc and s to skip it on purpose")
		}
		out = append(out, "**Answer:** "+answer)
	case "Affirm":
		if f.yes == nil {
			return nil, fmt.Errorf("y or n")
		}
		out = append(out, map[bool]string{true: "**Answer:** Yes.", false: "**Answer:** No."}[*f.yes])
	case "Select":
		for i, o := range f.prompt.Options {
			box := "[ ]"
			if f.checked[i] {
				box = "[x]"
			}
			out = append(out, "- "+box+" "+o)
		}
	default:
		return nil, fmt.Errorf("the TUI doesn't know how to answer a %s", f.prompt.Kind)
	}
	if note := strings.TrimSpace(string(f.note)); note != "" {
		if f.prompt.Kind == "Select" {
			// Directly under the list, markdown would make it the last option's.
			out = append(out, "")
		}
		out = append(out, "**Note:** "+note)
	}
	return out, nil
}

func (f *form) view(w int) string {
	var b strings.Builder
	verb := "Note on"
	switch {
	case f.prompt != nil:
		verb = "Answer"
	case f.request:
		verb = "Request to revisit, on"
	}
	b.WriteString(styles.title.Render(wrap(verb+" · "+f.about, w)) + "\n\n")
	if f.request {
		b.WriteString(styles.dim.Render(wrap("This goes in drafts/, as a request to revisit this plan. The plan stays as written, and a revisit answers the request.", w)) + "\n\n")
	}
	if f.first {
		b.WriteString(styles.kind("Changed").Render(wrap("This creates plan.draft.notes.md, which fixes the draft until it converts or the notes are deleted.", w)) + "\n\n")
	}
	box := func(label string, t []rune, focused bool) {
		text := string(t)
		if focused {
			text += "▏"
		}
		head := styles.dim.Render(label)
		if focused {
			head = styles.active.Render(label)
		}
		b.WriteString(head + "\n")
		// Wrapped, not cut off, so what's typed past the edge can be seen.
		for _, l := range strings.Split(wrap(text, w-2), "\n") {
			b.WriteString("  " + l + "\n")
		}
		b.WriteString("\n")
	}
	switch {
	case f.prompt == nil:
		box("comment", f.text, true)
	case f.prompt.Kind == "Question":
		box("answer", f.text, f.field == 0)
	case f.prompt.Kind == "Affirm":
		label := "answer: y or n"
		if f.yes != nil {
			label = map[bool]string{true: "answer: yes", false: "answer: no"}[*f.yes]
		}
		if f.field == 0 {
			b.WriteString(styles.active.Render(label) + "\n\n")
		} else {
			b.WriteString(styles.dim.Render(label) + "\n\n")
		}
	case f.prompt.Kind == "Select":
		b.WriteString(styles.dim.Render("space chooses; any number") + "\n")
		for i, o := range f.prompt.Options {
			mark := "[ ]"
			if f.checked[i] {
				mark = "[x]"
			}
			line := ansi.Truncate(mark+" "+o, w, "…")
			if f.field == 0 && i == f.optCur {
				line = styles.cursor.Render(line)
			}
			b.WriteString("  " + line + "\n")
		}
		b.WriteString("\n")
	}
	if f.prompt != nil {
		box("note (optional)", f.note, f.field == 1)
		b.WriteString(field("fallback", f.prompt.Fallback, w))
	}
	if f.err != "" {
		b.WriteString("\n" + styles.err.Render(wrap(f.err, w)) + "\n")
	}
	return b.String()
}

// ---- writing

func (m *Model) workDir() string {
	if m.dir != "" {
		return m.dir
	}
	d, _ := os.Getwd()
	return d
}

// openDraft finds the draft on disk and checks it's the one this page showed.
func (m *Model) openDraft(d *substrate.Doc) (*notes.Draft, error) {
	return notes.Open(m.workDir(), d.Draft.File, d.Draft.SHA256)
}

// write saves what the form says into the notes file, then reads the page
// again, so what shows is what phoropter made of it.
func (m *Model) write(s *docScreen, f *form, skip bool) tea.Cmd {
	open := m.openDraft
	if f.request {
		open = func(d *substrate.Doc) (*notes.Draft, error) { return notes.OpenRequest(m.workDir(), d.File) }
	}
	d, err := open(s.plan)
	if err != nil {
		m.status = err.Error()
		return nil
	}
	if f.prompt != nil {
		h, ok := d.Source.Prompt(f.prompt.Kind, f.prompt.Text)
		if !ok {
			m.status = "couldn't find this prompt's heading in " + d.Path
			return nil
		}
		lines := []string{"**Skipped.**"}
		if !skip {
			if lines, err = f.lines(); err != nil {
				f.err = err.Error()
				return nil
			}
		}
		d.Notes.Answer(d.Source, h, lines)
	} else {
		text := strings.TrimSpace(string(f.text))
		if text == "" {
			f.err = "write something, or esc"
			return nil
		}
		var entry *notes.Heading
		if f.entry >= 0 {
			if f.entry >= len(d.Source.Entries) || !sameEntry(d.Source.Entries[f.entry].Raw, s.plan.Entries[f.entry]) {
				m.status = "couldn't find this entry's heading in " + d.Path
				return nil
			}
			entry = &d.Source.Entries[f.entry]
		}
		d.Notes.Comment(d.Source, entry, m.author(), text)
	}
	if err := d.Save(); err != nil {
		m.status = err.Error()
		return nil
	}
	s.form = nil
	return m.reload()
}

// sameEntry checks the nth `## ` line in the source is the nth entry on the
// page: same kind, same title, once inline markdown is set aside.
func sameEntry(raw string, e substrate.Entry) bool {
	flat := func(s string) string {
		return strings.Join(strings.Fields(strings.NewReplacer("`", "", "*", "", "_", "").Replace(s)), " ")
	}
	return flat(raw) == flat("## "+e.Kind+": "+e.Title)
}

func (m *Model) author() string {
	if m.authorName == "" {
		m.authorName = notes.Author()
	}
	return m.authorName
}

type editedMsg struct {
	field *[]rune
	text  string
	err   error
}

// editor opens $VISUAL or $EDITOR on what's typed so far, and takes back what
// was saved.
func editor(field *[]rune) tea.Cmd {
	tmp, err := os.CreateTemp("", "phoropter-tui-*.md")
	if err != nil {
		return func() tea.Msg { return editedMsg{err: err} }
	}
	tmp.WriteString(string(*field))
	tmp.Close()
	name := os.Getenv("VISUAL")
	if name == "" {
		name = os.Getenv("EDITOR")
	}
	if name == "" {
		name = "vi"
	}
	args := append(strings.Fields(name), tmp.Name())
	return tea.ExecProcess(exec.Command(args[0], args[1:]...), func(err error) tea.Msg {
		defer os.Remove(tmp.Name())
		b, rerr := os.ReadFile(tmp.Name())
		if err == nil {
			err = rerr
		}
		return editedMsg{field: field, text: strings.TrimRight(string(b), "\n"), err: err}
	})
}
