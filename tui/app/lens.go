package app

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/hypertheory-labs/phoropter/tui/notes"
	"github.com/hypertheory-labs/phoropter/tui/substrate"
)

// Lenses (percale): L from the plan list opens a reader's lenses, a lens can be
// filtered by kind, and + on an entry or a plan adds it to one. Filtering a
// page already read is a local fact; what each item is comes from the page.

// ---- the list of lenses

type lensesScreen struct {
	lenses *substrate.Lenses
	cur    int
}

func (s *lensesScreen) title() string { return "Lenses" }
func (s *lensesScreen) help() string {
	return "j/k move · enter open · y copy link · b browser · r refresh · esc back"
}
func (s *lensesScreen) refresh() ([]string, string) { return []string{s.lenses.URL}, s.lenses.URL }

func (s *lensesScreen) here() (string, string) {
	if s.cur < len(s.lenses.Lenses) {
		return s.lenses.Lenses[s.cur].URL, ""
	}
	return s.lenses.URL, ""
}

func (s *lensesScreen) key(m *Model, k string) tea.Cmd {
	switch k {
	case "enter", "l", "right":
		if s.cur < len(s.lenses.Lenses) {
			return m.open(s.lenses.Lenses[s.cur].URL)
		}
	case "esc", "backspace", "h", "left":
		m.back()
	default:
		if n := len(s.lenses.Lenses); n > 0 {
			s.cur = move(s.cur, n, k)
		}
	}
	return nil
}

func (s *lensesScreen) view(m *Model, w, h int) string {
	lw := min(max(w*2/5, 30), w/2)
	var rows []string
	for i, l := range s.lenses.Lenses {
		text := ansi.Truncate(fmt.Sprintf("%s %s", l.Title, styles.dim.Render(plural(l.Items, "item"))), lw, "…")
		if i == s.cur {
			text = styles.cursor.Render(ansi.Strip(text))
		}
		rows = append(rows, text)
	}
	rw := detailWidth(w)
	var b strings.Builder
	if len(s.lenses.Lenses) == 0 {
		b.WriteString(wrap("No lenses yet. + on an entry or a plan starts one, and the agent can write them: drafts/lenses/<name>.md, one reference per line.", rw))
	} else {
		l := s.lenses.Lenses[s.cur]
		b.WriteString(styles.title.Render(wrap(l.Title, rw)) + "\n")
		b.WriteString(styles.dim.Render(l.File+" · "+plural(l.Items, "item")) + "\n")
	}
	return split(listWindow(rows, s.cur, h), b.String(), w, h, 0)
}

// ---- one lens

// filters are what f cycles through: everything, then one kind, then the
// guesses still open (unanswered, or answered not tested).
var filters = []string{"", "Decision", "Decline", "open guesses"}

type lensScreen struct {
	lens   *substrate.Lens
	filter int
	cur    int // into shown()
	scroll int
}

func (s *lensScreen) title() string { return "lens: " + s.lens.Title }
func (s *lensScreen) help() string {
	return "j/k move · f filter · enter open · y/Y copy link/ref · b browser · J/K scroll · esc back"
}
func (s *lensScreen) refresh() ([]string, string) { return []string{s.lens.URL}, s.lens.URL }

func (s *lensScreen) shown() []substrate.LensItem {
	var out []substrate.LensItem
	for _, it := range s.lens.Items {
		switch f := filters[s.filter]; f {
		case "":
			out = append(out, it)
		case "open guesses":
			if it.Kind == "Guess" && it.Status == "open" {
				out = append(out, it)
			}
		default:
			if it.Kind == f {
				out = append(out, it)
			}
		}
	}
	return out
}

func (s *lensScreen) here() (string, string) {
	items := s.shown()
	if s.cur < len(items) && items[s.cur].URL != "" {
		return items[s.cur].URL, items[s.cur].Reference
	}
	return s.lens.URL, ""
}

func (s *lensScreen) key(m *Model, k string) tea.Cmd {
	items := s.shown()
	switch k {
	case "f":
		s.filter = (s.filter + 1) % len(filters)
		s.cur, s.scroll = 0, 0
	case "enter", "l", "right":
		if s.cur < len(items) {
			if items[s.cur].URL == "" {
				m.status = "this line doesn't resolve: " + items[s.cur].Problem
				return nil
			}
			return m.open(items[s.cur].URL)
		}
	case "J", "ctrl+d", "pgdown":
		s.scroll += 5
	case "K", "ctrl+u", "pgup":
		s.scroll = max(0, s.scroll-5)
	case "esc", "backspace", "h", "left":
		m.back()
	default:
		if len(items) > 0 {
			s.cur = move(s.cur, len(items), k)
			s.scroll = 0
		}
	}
	return nil
}

func (s *lensScreen) view(m *Model, w, h int) string {
	lw := min(max(w*2/5, 30), w/2)
	items := s.shown()
	label := "all"
	if f := filters[s.filter]; f != "" {
		label = f
	}
	rows := []string{styles.dim.Render(fmt.Sprintf("%s · %d of %d", label, len(items), len(s.lens.Items)))}
	for i, it := range items {
		var text string
		switch {
		case it.Problem != "":
			text = styles.dim.Render("✗ " + it.Reference)
		default:
			text = styles.kind(it.Kind).Render(it.Kind) + " " + it.Title
			if it.Status != "" {
				text += " " + styles.verdict(it.Status).Render("("+it.Status+")")
			}
		}
		text = ansi.Truncate(text, lw, "…")
		if i == s.cur {
			text = styles.cursor.Render(ansi.Strip(text))
		}
		rows = append(rows, text)
	}
	rw := detailWidth(w)
	var b strings.Builder
	b.WriteString(styles.title.Render(wrap(s.lens.Title, rw)) + "\n")
	b.WriteString(styles.dim.Render(s.lens.File) + "\n")
	if s.lens.RepositoryNote != "" {
		b.WriteString(wrap(s.lens.RepositoryNote, rw) + "\n")
	}
	if s.lens.Intro != "" {
		b.WriteString("\n" + wrap(s.lens.Intro, rw) + "\n")
	}
	b.WriteString("\n")
	if s.cur < len(items) {
		it := items[s.cur]
		if it.Problem != "" {
			b.WriteString(field("doesn't resolve", it.Problem, rw))
		} else {
			b.WriteString(styles.kind(it.Kind).Render(it.Kind) + "  " + wrap(it.Title, rw-len(it.Kind)-2) + "\n")
			b.WriteString(styles.dim.Render("in "+it.Plan) + "\n\n")
		}
		if it.Note != "" {
			b.WriteString(field("note", it.Note, rw))
		}
		b.WriteString(field("reference", it.Reference, rw))
	} else if len(s.lens.Items) > 0 {
		b.WriteString(styles.dim.Render("nothing in this lens is "+label+" · f for the next filter") + "\n")
	}
	detail := b.String()
	s.scroll = min(s.scroll, max(0, strings.Count(detail, "\n")+1-h))
	return split(listWindow(rows, s.cur+1, h), detail, w, h, s.scroll)
}

// ---- + adds what you're on to a lens

// lensPicker asks only which lens: collecting comes first, notes later (Jeff).
type lensPicker struct {
	ref    string
	first  string // the repository's first commit, for a new lens's header
	lenses []substrate.LensSummary
	text   []rune
	tab    int
	err    string
}

// collect opens the picker for the current row, offering the last lens used.
func (s *docScreen) collect(m *Model) {
	r := s.rows[s.cur]
	ref := reference(r.doc, r.entry)
	switch {
	case r.doc.Draft != nil:
		m.status = "a draft's entries go in a lens once the draft lands"
		return
	case r.prompt != nil || r.note != nil || ref == "":
		m.status = "+ adds an entry or a whole plan to a lens"
		return
	}
	p, err := m.c.Page(r.doc.URL)
	if err != nil || p.LensesURL == "" {
		m.status = "lenses need phoropter's dev server, which reads drafts/lenses/"
		return
	}
	m.c.Forget(m.c.Resolve(p.LensesURL))
	ls, err := m.c.Lenses(p.LensesURL)
	if err != nil {
		m.status = err.Error()
		return
	}
	s.adding = &lensPicker{ref: ref, first: ls.FirstCommit, lenses: ls.Lenses, text: []rune(m.lastLens), tab: -1}
}

func (s *docScreen) captureLens(m *Model, msg tea.KeyMsg) tea.Cmd {
	p := s.adding
	switch msg.Type {
	case tea.KeyEsc:
		s.adding = nil
	case tea.KeyEnter:
		name := strings.TrimSpace(string(p.text))
		for _, l := range p.lenses {
			// A lens chosen by its title goes in its own file, whatever its name.
			if strings.EqualFold(l.Title, name) {
				name = strings.TrimSuffix(filepath.Base(l.File), ".md")
			}
		}
		path, added, err := notes.AppendLens(m.workDir(), name, p.ref, p.first)
		if err != nil {
			p.err = err.Error()
			return nil
		}
		m.lastLens = strings.TrimSpace(string(p.text))
		s.adding = nil
		where := filepath.Join("drafts", "lenses", filepath.Base(path))
		if added {
			m.status = "added to " + where
		} else {
			m.status = "already in " + where
		}
	case tea.KeyTab:
		if len(p.lenses) > 0 {
			p.tab = (p.tab + 1) % len(p.lenses)
			p.text = []rune(p.lenses[p.tab].Title)
		}
	case tea.KeyBackspace:
		if len(p.text) > 0 {
			p.text = p.text[:len(p.text)-1]
		}
	case tea.KeyRunes, tea.KeySpace:
		p.text = append(p.text, msg.Runes...)
	}
	return nil
}

func (p *lensPicker) view(w int) string {
	var b strings.Builder
	b.WriteString(styles.title.Render("Add to a lens") + "\n\n")
	b.WriteString(wrap(p.ref, w) + "\n\n")
	b.WriteString("lens: " + string(p.text) + "█\n\n")
	if len(p.lenses) > 0 {
		b.WriteString(styles.dim.Render("tab cycles your lenses; a new name starts one") + "\n")
		for i, l := range p.lenses {
			line := ansi.Truncate("  "+l.Title+" "+styles.dim.Render(plural(l.Items, "item")), w, "…")
			if i == p.tab {
				line = styles.cursor.Render(ansi.Strip(line))
			}
			b.WriteString(line + "\n")
		}
	} else {
		b.WriteString(styles.dim.Render("no lenses yet; a name starts one in drafts/lenses/") + "\n")
	}
	if p.err != "" {
		b.WriteString("\n" + p.err + "\n")
	}
	b.WriteString("\n" + styles.dim.Render("enter add · esc cancel"))
	return b.String()
}
