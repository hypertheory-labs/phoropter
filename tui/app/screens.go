package app

import (
	"fmt"
	"path"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/hypertheory-labs/phoropter/tui/notes"
	"github.com/hypertheory-labs/phoropter/tui/substrate"
)

// split lays out a list on the left and a detail pane on the right.
func split(left, right string, w, h, rightScroll int) string {
	lw := min(max(w*2/5, 30), w/2)
	rw := w - lw - 3
	bar := styles.rule.Render(strings.TrimSuffix(strings.Repeat(" │ \n", h), "\n"))
	return lipgloss.JoinHorizontal(lipgloss.Top, clip(left, lw, h, 0), bar, clip(right, rw, h, rightScroll))
}

func detailWidth(w int) int { return w - min(max(w*2/5, 30), w/2) - 3 }

// listWindow scrolls a list so the cursor stays on screen.
func listWindow(rows []string, cur, h int) string {
	start := 0
	if cur >= h {
		start = cur - h + 1
	}
	end := min(len(rows), start+h)
	return strings.Join(rows[start:end], "\n")
}

func move(cur, n int, k string) int {
	switch k {
	case "j", "down":
		return min(cur+1, n-1)
	case "k", "up":
		return max(cur-1, 0)
	case "home":
		return 0
	case "end", "G":
		return n - 1
	case "tab":
		return (cur + 1) % n
	case "shift+tab":
		return (cur + n - 1) % n
	}
	return cur
}

// ---- the plan list

type listScreen struct {
	home   *substrate.Home
	draft  *substrate.Doc // the draft's page, when the home page links one
	onDisk bool           // no draft linked, but plan.draft.md is in this repo
	oldest bool           // newest first unless flipped
	cur    int            // 0 is the draft's slot; the plans follow
}

func newListScreen(h *substrate.Home, draft *substrate.Doc, onDisk bool) *listScreen {
	s := &listScreen{home: h, draft: draft, onDisk: onDisk}
	if draft == nil && len(h.Plans) > 0 {
		s.cur = 1
	}
	return s
}

// plan is the plan at a position in the list, in the order shown.
func (s *listScreen) plan(i int) substrate.PlanSummary {
	if s.oldest {
		return s.home.Plans[i]
	}
	return s.home.Plans[len(s.home.Plans)-1-i]
}

func (s *listScreen) title() string {
	return fmt.Sprintf("%s at %s", s.home.Repository, s.home.Commit)
}

func (s *listScreen) help() string {
	return "j/k move · enter open · o oldest/newest first · y copy link · b browser · r refresh"
}

func (s *listScreen) refresh() ([]string, string) {
	return []string{s.home.URL, s.home.DraftURL}, s.home.URL
}

func (s *listScreen) key(m *Model, k string) tea.Cmd {
	switch k {
	case "o":
		// The same plan stays chosen.
		if s.cur > 0 {
			s.cur = len(s.home.Plans) + 1 - s.cur
		}
		s.oldest = !s.oldest
		return nil
	case "enter", "l", "right":
		if s.cur == 0 {
			if s.draft == nil {
				m.status = "there's no draft to open"
				return nil
			}
			return m.open(s.home.DraftURL)
		}
		return m.open(s.plan(s.cur - 1).URL)
	}
	s.cur = move(s.cur, len(s.home.Plans)+1, k)
	return nil
}

func openPrompts(d *substrate.Doc) int {
	n := 0
	for _, p := range d.Prompts {
		if p.Answer == "" {
			n++
		}
	}
	return n
}

// slot is the draft's place at the top of the list, there with or without one.
func (s *listScreen) slot(w int) string {
	var head, sub string
	switch {
	case s.draft != nil:
		head = "✎ " + s.draft.Title
		sub = "draft, read at " + readAt(s.draft.Draft)
		if n := openPrompts(s.draft); n > 0 {
			sub += " · " + plural(n, "open prompt")
		}
	case s.onDisk:
		head = "✎ There's a draft on disk"
		sub = "this site doesn't serve drafts; phoropter's dev server does"
	default:
		head = "No draft"
		sub = "when plan.draft.md exists, the dev server shows it here"
	}
	inner := w - 4
	head = ansi.Truncate(head, inner, "…")
	if s.cur == 0 {
		head = styles.cursor.Render(head)
	} else if s.draft != nil {
		head = styles.title.Render(head)
	}
	box := styles.slot
	if s.draft == nil {
		box = styles.slotEmpty
	}
	return box.Width(w - 2).Render(head + "\n" + styles.dim.Render(ansi.Truncate(sub, inner, "…")))
}

// meta is a plan's dim line: what the page keeps in its metadata.
func meta(p substrate.PlanSummary) string {
	// Notes are someone saying the plan matters now, so they come right after the name.
	parts := []string{p.Name}
	if p.NoteCount > 0 {
		parts = append(parts, "✎ "+plural(p.NoteCount, "note"))
	}
	parts = append(parts, p.Date, plural(p.Entries, "entry"))
	if p.AsBuiltURL != "" {
		parts = append(parts, "✓ as-built")
	}
	if p.RevisitsURL != "" {
		parts = append(parts, "revisits "+path.Base(strings.TrimSuffix(p.RevisitsURL, "/")))
	}
	return strings.Join(parts, " · ")
}

func (s *listScreen) view(m *Model, w, h int) string {
	lw := min(max(w*2/5, 30), w/2)
	slot := s.slot(lw)
	order := "newest first"
	if s.oldest {
		order = "oldest first"
	}
	left := []string{slot, styles.dim.Render(ansi.Truncate(fmt.Sprintf("Plans · %s · o flips", order), lw, "…"))}
	if len(s.home.Plans) == 0 {
		left = append(left, styles.dim.Render("No plans on this site yet."))
	}
	// A box a plan, with a gap after; scroll so the chosen one shows.
	const each = 5
	room := max(1, (h-lipgloss.Height(slot)-1)/each)
	first := 0
	if s.cur > room {
		first = s.cur - room
	}
	inner := lw - 4
	for i := first; i < len(s.home.Plans) && i < first+room; i++ {
		p := s.plan(i)
		title, box := ansi.Truncate(p.Title, inner, "…"), styles.card
		if i+1 == s.cur {
			title, box = styles.cursor.Render(title), styles.cardChosen
		}
		left = append(left, box.Width(lw-2).Render(title+"\n"+styles.dim.Render(ansi.Truncate(meta(p), inner, "…"))), "")
	}

	rw := detailWidth(w)
	var b strings.Builder
	if s.cur == 0 {
		switch {
		case s.draft != nil:
			d := s.draft
			b.WriteString(styles.title.Render(wrap(d.Title, rw)) + "\n\n")
			b.WriteString(field("read at", readAt(d.Draft), rw))
			b.WriteString(field("anchors at", d.AnchorsAt.Label, rw))
			b.WriteString(field("entries", fmt.Sprint(len(d.Entries)), rw))
			b.WriteString(field("prompts", fmt.Sprintf("%d, %d open", len(d.Prompts), openPrompts(d)), rw))
			b.WriteString(field("notes", fmt.Sprint(len(d.Annotations)), rw))
			if d.Intro != "" {
				b.WriteString("\n" + wrap(d.Intro, rw) + "\n")
			}
			b.WriteString("\n" + styles.dim.Render(wrap("The repository's one draft, plan.draft.md, read from the working tree on every request. Nothing in it is testimony yet. enter opens it.", rw)))
		case s.onDisk:
			b.WriteString(styles.title.Render("A draft this site doesn't show") + "\n\n" + wrap("plan.draft.md exists in this repository, but a built site is a function of one commit and never holds a draft. Phoropter's dev server shows it: node src/serve.ts <repo>, from phoropter.", rw))
		default:
			b.WriteString(styles.title.Render("No draft") + "\n\n" + wrap("A repository has at most one draft, plan.draft.md at its root. Most plans never are one. A draft is for when the shared understanding is thin; when it exists, phoropter's dev server shows it, and this slot opens it.", rw))
		}
		return split(strings.Join(left, "\n"), b.String(), w, h, 0)
	}
	p := s.plan(s.cur - 1)
	b.WriteString(styles.title.Render(wrap(p.Title, rw)) + "\n\n")
	b.WriteString(field("name", p.Name, rw))
	b.WriteString(field("date", p.Date, rw))
	b.WriteString(field("added", p.Added.Label, rw))
	b.WriteString(field("base", p.Base.Label, rw))
	b.WriteString(field("entries", fmt.Sprint(p.Entries), rw))
	if p.AsBuiltURL != "" {
		b.WriteString(field("as-built", "yes", rw))
	} else {
		b.WriteString(field("as-built", styles.dim.Render("none yet"), rw))
	}
	if p.RevisitsURL != "" {
		b.WriteString(field("revisits", path.Base(strings.TrimSuffix(p.RevisitsURL, "/")), rw))
	}
	b.WriteString("\n" + styles.dim.Render(wrap("Ordered by the commit that added each plan, "+order+".", rw)))
	return split(strings.Join(left, "\n"), b.String(), w, h, 0)
}

func readAt(d *substrate.Draft) string {
	if t, err := time.Parse(time.RFC3339, d.ReadAt); err == nil {
		return t.UTC().Format("15:04 UTC")
	}
	return d.ReadAt
}

// ---- a plan and its as-built

type row struct {
	doc    *substrate.Doc
	entry  *substrate.Entry // nil for the plan's own "about" row
	prompt *substrate.Prompt
	note   *substrate.Annotation
}

type docScreen struct {
	plan, asBuilt *substrate.Doc
	rows          []row
	cur           int
	anchor        int
	layer         int // index into layers
	scroll        int
	menu          []menuItem
	menuOpen      bool
	menuCur       int
	form          *form // an answer or note being written, on a draft
}

func (s *docScreen) capturing() bool { return s.form != nil }

func (s *docScreen) capture(m *Model, msg tea.KeyMsg) tea.Cmd {
	write, cancel, edit := s.form.key(msg)
	switch {
	case cancel:
		s.form = nil
	case write:
		return m.write(s, s.form, false)
	case edit != nil:
		return editor(edit)
	}
	return nil
}

// planNote opens a note on a landed plan, kept in drafts/ (tarlatan): someone
// saying the plan matters now (organza). A plan takes notes, never edits.
func (s *docScreen) planNote(m *Model, r row) {
	if r.doc.IsAsBuilt {
		m.status = "an as-built takes no notes; note its plan"
		return
	}
	if r.note != nil {
		m.status = "a note can't be noted; notes go on entries or the whole plan"
		return
	}
	if _, err := notes.OpenPlanNotes(m.workDir(), r.doc.File); err != nil {
		m.status = err.Error()
		return
	}
	var f *form
	if r.entry != nil {
		for i := range r.doc.Entries {
			if &r.doc.Entries[i] == r.entry {
				f = newNoteForm(i, r.entry.Kind+": "+r.entry.Title, false)
			}
		}
	} else {
		f = newNoteForm(-1, "the whole plan", false)
	}
	f.landed = true
	s.form = f
}

// annotate opens a form for the current row: a prompt's answer, a note on an
// entry, or on the About row a note on the whole draft.
func (s *docScreen) annotate(m *Model) {
	r := s.rows[s.cur]
	if r.doc.Draft == nil {
		s.planNote(m, r)
		return
	}
	d, err := m.openDraft(r.doc)
	if err != nil {
		m.status = err.Error()
		return
	}
	switch {
	case r.prompt != nil:
		s.form = newPromptForm(r.prompt, !d.HasNotes)
	case r.entry != nil:
		for i := range r.doc.Entries {
			if &r.doc.Entries[i] == r.entry {
				s.form = newNoteForm(i, r.entry.Kind+": "+r.entry.Title, !d.HasNotes)
			}
		}
	case r.isAbout():
		s.form = newNoteForm(-1, "the whole draft", !d.HasNotes)
	default:
		m.status = "a note can't be noted; notes go on entries, prompts, or the whole draft"
	}
}

type menuItem struct {
	rel, label, url string
}

// layers are the profile's view roles; layerNames are what a person reads.
var (
	layers     = []string{"base", "as-built head", "generated"}
	layerNames = []string{"before", "as built", "now"}
)

func newDocScreen(plan, asBuilt *substrate.Doc) *docScreen {
	s := &docScreen{plan: plan, asBuilt: asBuilt}
	s.rows = append(s.rows, row{doc: plan})
	s.rows = append(s.rows, draftRows(plan, "")...)
	for i := range plan.Entries {
		s.rows = append(s.rows, row{doc: plan, entry: &plan.Entries[i]})
		s.rows = append(s.rows, draftRows(plan, plan.Entries[i].ID)...)
	}
	if asBuilt != nil {
		s.rows = append(s.rows, row{doc: asBuilt})
		for i := range asBuilt.Entries {
			s.rows = append(s.rows, row{doc: asBuilt, entry: &asBuilt.Entries[i]})
		}
	}
	return s
}

// draftRows are a draft's prompts, then its notes, that follow the entry with
// this ID ("" for those before every entry).
func draftRows(d *substrate.Doc, after string) []row {
	var out []row
	for i := range d.Prompts {
		if d.Prompts[i].After == after {
			out = append(out, row{doc: d, prompt: &d.Prompts[i]})
		}
	}
	for i := range d.Annotations {
		if d.Annotations[i].After == after {
			out = append(out, row{doc: d, note: &d.Annotations[i]})
		}
	}
	return out
}

func (r row) isAbout() bool { return r.entry == nil && r.prompt == nil && r.note == nil }

// selectURL puts the cursor on the entry a URL's fragment names.
func (s *docScreen) selectURL(u string) {
	page, frag := substrate.Unfragment(u), substrate.Fragment(u)
	for i, r := range s.rows {
		if r.doc.URL != page {
			continue
		}
		if (frag == "" && r.isAbout()) || (r.entry != nil && r.entry.ID == frag) {
			s.cur = i
			s.settle()
			return
		}
	}
}

// settle keeps the chosen layer when the entry has it, and otherwise moves to
// the first layer it has.
func (s *docScreen) settle() {
	s.anchor, s.scroll = 0, 0
	a := s.currentAnchor()
	if a == nil || viewAt(a, layers[s.layer]) != nil {
		return
	}
	for i, l := range layers {
		if viewAt(a, l) != nil {
			s.layer = i
			return
		}
	}
}

func (s *docScreen) currentAnchor() *substrate.Anchor {
	e := s.rows[s.cur].entry
	if e == nil || len(e.Anchors) == 0 {
		return nil
	}
	return &e.Anchors[s.anchor%len(e.Anchors)]
}

func viewAt(a *substrate.Anchor, role string) *substrate.View {
	for i := range a.Views {
		if a.Views[i].Role == role {
			return &a.Views[i]
		}
	}
	return nil
}

func (s *docScreen) title() string {
	d := s.plan.Draft
	if d == nil {
		return s.plan.Title
	}
	return fmt.Sprintf("draft: %s · not committed, read at %s · anchors at %s", s.plan.Title, readAt(d), s.plan.AnchorsAt)
}

func (s *docScreen) accent() bool { return s.plan.Draft != nil }

func (s *docScreen) help() string {
	if s.form != nil {
		return s.form.help()
	}
	if s.menuOpen {
		return "j/k choose · enter follow · esc close"
	}
	if s.plan.Draft != nil {
		return "j/k or tab move · a answer or note · s skip · n anchor · 1 2 3 layer · enter links · r refresh · esc back"
	}
	return "j/k or tab entry · a note · n anchor · 1 2 3 [ ] layer · enter links · y/Y copy link/ref · b browser · esc back"
}

// refresh reads the plan and its as-built again, back at the same entry.
func (s *docScreen) refresh() ([]string, string) {
	forget := []string{s.plan.URL}
	if s.asBuilt != nil {
		forget = append(forget, s.asBuilt.URL)
	}
	r := s.rows[s.cur]
	at := r.doc.URL
	switch {
	case r.entry != nil:
		at += "#" + r.entry.ID
	case r.prompt != nil && r.prompt.After != "":
		at += "#" + r.prompt.After
	case r.note != nil && r.note.After != "":
		at += "#" + r.note.After
	}
	return forget, at
}

func (s *docScreen) key(m *Model, k string) tea.Cmd {
	if s.menuOpen {
		switch k {
		case "esc", "backspace", "h", "left":
			s.menuOpen = false
		case "enter", "l", "right":
			s.menuOpen = false
			return m.open(s.menu[s.menuCur].url)
		default:
			s.menuCur = move(s.menuCur, len(s.menu), k)
		}
		return nil
	}
	switch k {
	case "j", "down", "k", "up", "home", "end", "G", "tab", "shift+tab":
		s.cur = move(s.cur, len(s.rows), k)
		s.settle()
	case "n":
		if e := s.rows[s.cur].entry; e != nil && len(e.Anchors) > 0 {
			s.anchor = (s.anchor + 1) % len(e.Anchors)
			s.scroll = 0
		}
	case "N":
		if e := s.rows[s.cur].entry; e != nil && len(e.Anchors) > 0 {
			s.anchor = (s.anchor + len(e.Anchors) - 1) % len(e.Anchors)
			s.scroll = 0
		}
	case "1", "2", "3":
		s.layer = int(k[0] - '1')
	case "]":
		s.layer = (s.layer + 1) % len(layers)
	case "[":
		s.layer = (s.layer + len(layers) - 1) % len(layers)
	case "J", "ctrl+d", "pgdown":
		s.scroll += 5
	case "K", "ctrl+u", "pgup":
		s.scroll = max(0, s.scroll-5)
	case "enter", "l", "right":
		s.menu = s.links()
		if len(s.menu) == 0 {
			m.status = "nothing links out from here"
			return nil
		}
		s.menuOpen, s.menuCur = true, 0
	case "a":
		s.annotate(m)
	case "s":
		if r := s.rows[s.cur]; r.prompt != nil && r.doc.Draft != nil {
			return m.write(s, newPromptForm(r.prompt, false), true)
		}
	case "esc", "backspace", "h", "left":
		m.back()
	}
	return nil
}

// links are every rel out of the current row, labeled with what following
// it answers.
func (s *docScreen) links() []menuItem {
	r := s.rows[s.cur]
	var out []menuItem
	if r.prompt != nil || r.note != nil {
		return nil
	}
	if r.entry == nil {
		d := r.doc
		if d.Revisits != nil {
			out = append(out, menuItem{"revisits", "stands on: " + d.Revisits.Label, d.Revisits.URL})
		}
		for _, l := range d.RevisitedBy {
			out = append(out, menuItem{"revisited-by", "revisited by: " + l.Label, l.URL})
		}
		return out
	}
	e := r.entry
	for _, a := range e.Anchors {
		if a.Place != nil {
			out = append(out, menuItem{"anchored-at", "every plan at " + a.Where, a.Place.URL})
		}
		if a.Target != nil {
			out = append(out, menuItem{"anchored-at", "points at: " + a.Target.Label, a.Target.URL})
		}
	}
	if e.Answers != nil {
		out = append(out, menuItem{"answers", "the guess this answers", e.Answers.URL})
	}
	for _, l := range e.AnsweredBy {
		out = append(out, menuItem{"answered-by", l.Label, l.URL})
	}
	for _, l := range e.AnchoredBy {
		out = append(out, menuItem{"anchored-by", "pointed at by: " + l.Label, l.URL})
	}
	return out
}

func (s *docScreen) view(m *Model, w, h int) string {
	lw := min(max(w*2/5, 30), w/2)
	var rows []string
	for i, r := range s.rows {
		var text string
		switch {
		case r.prompt != nil:
			text = "    ? " + r.prompt.Kind + ": " + r.prompt.Text
			if r.prompt.Answer != "" {
				text = "    ✓ " + r.prompt.Kind + ": " + r.prompt.Text
			}
		case r.note != nil:
			text = "    ✎ " + r.note.Heading
		case r.entry == nil && r.doc.IsAsBuilt:
			text = "── As built, head " + r.doc.Head.Label
		case r.entry == nil && r.doc.Draft != nil:
			text = "── About this draft · a to note it"
		case r.entry == nil && !r.doc.IsAsBuilt:
			text = "── About this plan · a to note it"
		case r.entry == nil:
			text = "── About this plan"
		default:
			text = "  " + styles.kind(r.entry.Kind).Render(r.entry.Kind) + " " + r.entry.Title
			if r.entry.Verdict != "" {
				text += " " + styles.verdict(r.entry.Verdict).Render("("+r.entry.Verdict+")")
			}
		}
		text = ansi.Truncate(text, lw, "…")
		if i == s.cur {
			text = styles.cursor.Render(ansi.Strip(text))
		} else if r.isAbout() {
			text = styles.dim.Render(ansi.Strip(text))
		}
		rows = append(rows, text)
	}
	rw := detailWidth(w)
	detail := s.detail(rw)
	switch {
	case s.form != nil:
		detail = s.form.view(rw)
	case s.menuOpen:
		detail = s.menuView(rw)
	}
	lines := strings.Count(detail, "\n") + 1
	s.scroll = min(s.scroll, max(0, lines-h))
	return split(listWindow(rows, s.cur, h), detail, w, h, s.scroll)
}

func (s *docScreen) menuView(w int) string {
	var b strings.Builder
	b.WriteString(styles.title.Render("Follow a link") + "\n\n")
	for i, it := range s.menu {
		line := ansi.Truncate(fmt.Sprintf("%-13s %s", it.rel, it.label), w, "…")
		if i == s.menuCur {
			line = styles.cursor.Render(ansi.Strip(line))
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}

func (s *docScreen) detail(w int) string {
	r := s.rows[s.cur]
	var b strings.Builder
	if r.prompt != nil {
		return promptDetail(r.prompt, w)
	}
	if r.note != nil {
		return noteDetail(r.note, w)
	}
	if r.entry == nil {
		d := r.doc
		b.WriteString(styles.title.Render(wrap(d.Title, w)) + "\n\n")
		for _, f := range d.Facts {
			b.WriteString(field(f[0], f[1], w))
		}
		if d.Intro != "" {
			b.WriteString("\n" + wrap(d.Intro, w) + "\n")
		}
		if len(d.Entries) == 0 {
			b.WriteString("\n" + styles.dim.Render("No entries.") + "\n")
		}
		switch {
		case d.Draft != nil:
			b.WriteString("\n" + styles.dim.Render(wrap("a here writes a note on the whole draft. On an entry, a notes that entry; on a prompt, it answers.", w)) + "\n")
		case !d.IsAsBuilt:
			b.WriteString("\n" + styles.dim.Render(wrap("A landed plan is never edited, but it takes notes. a here writes a note on the whole plan; on an entry, a note on that entry. Notes go in drafts/: to say the plan matters to what we're doing now, or that its ground has moved.", w)) + "\n")
		}
		return b.String()
	}
	e := r.entry
	b.WriteString(styles.kind(e.Kind).Render(e.Kind+":") + " " + styles.title.Render(e.Title) + "\n")
	if e.Verdict != "" {
		b.WriteString("verdict " + styles.verdict(e.Verdict).Render(e.Verdict) + "\n")
	}
	b.WriteString("\n" + wrap(e.Reason, w) + "\n")
	var rels []string
	if e.Answers != nil {
		rels = append(rels, "answers a guess")
	}
	if n := len(e.AnsweredBy); n > 0 {
		rels = append(rels, plural(n, "answer"))
	}
	if n := len(e.AnchoredBy); n > 0 {
		rels = append(rels, fmt.Sprintf("pointed at by %s", plural(n, "entry")))
	}
	if len(rels) > 0 {
		b.WriteString("\n" + styles.dim.Render(strings.Join(rels, " · ")+" · enter to follow") + "\n")
	}
	if len(e.Anchors) == 0 {
		b.WriteString("\n" + styles.dim.Render("No anchor: this entry doesn't say what code it's about.") + "\n")
		return b.String()
	}
	a := s.currentAnchor()
	head := fmt.Sprintf("── anchor %d of %d ", s.anchor%len(e.Anchors)+1, len(e.Anchors))
	b.WriteString("\n" + styles.rule.Render(head+strings.Repeat("─", max(0, w-len([]rune(head))))) + "\n")
	label := "where"
	if a.New {
		label = "where (new)"
	}
	b.WriteString(field(label, a.Where, w))
	if a.Quote != "" {
		b.WriteString(field("quote", a.Quote, w))
	}
	if a.Target != nil {
		b.WriteString(field("entry", a.Target.Label, w))
	}
	if a.Note != "" {
		b.WriteString(field("note", a.Note, w))
	}
	if len(a.Views) == 0 && a.Note != "" {
		// The note already says why there's nothing at any layer.
		return b.String()
	}
	b.WriteString("\n" + s.lensBar(a) + "\n")
	b.WriteString(viewBody(a, layers[s.layer], w))
	return b.String()
}

func promptDetail(p *substrate.Prompt, w int) string {
	var b strings.Builder
	b.WriteString(styles.title.Render(p.Kind+":") + " " + wrap(p.Text, w-len(p.Kind)-2) + "\n\n")
	for _, o := range p.Options {
		b.WriteString("  • " + o + "\n")
	}
	if len(p.Options) > 0 {
		b.WriteString("\n")
	}
	b.WriteString(field("fallback", p.Fallback, w))
	if p.Answer != "" {
		b.WriteString(field("answer", p.Answer, w))
	} else {
		b.WriteString("\n" + styles.dim.Render(wrap("Not answered yet. If it stays that way, the fallback is the decision.", w)) + "\n")
	}
	return b.String()
}

func noteDetail(n *substrate.Annotation, w int) string {
	var b strings.Builder
	b.WriteString(styles.title.Render("Note: "+n.Heading) + "\n\n")
	if n.Quote != "" {
		b.WriteString(field("quote", n.Quote, w))
	}
	b.WriteString("\n" + wrap(n.Comment, w) + "\n")
	return b.String()
}

// field is a dim label and a value wrapped beside it, continuation lines
// indented under the value.
func field(label, value string, w int) string {
	lines := strings.Split(wrap(value, w-12), "\n")
	for i := 1; i < len(lines); i++ {
		lines[i] = strings.Repeat(" ", 12) + lines[i]
	}
	return styles.dim.Render(fmt.Sprintf("%-11s ", label)) + strings.Join(lines, "\n") + "\n"
}

// lensBar shows the three layers, the chosen one lit, and each one's status.
func (s *docScreen) lensBar(a *substrate.Anchor) string {
	var parts []string
	for i, role := range layers {
		v := viewAt(a, role)
		text := fmt.Sprintf("%d %s", i+1, layerNames[i])
		if v != nil {
			text += " " + v.Commit.Label + " " + statusMarks[v.Status]
			if v.SameAs != "" {
				text += " =" + v.SameAs
			}
		}
		switch {
		case i == s.layer:
			parts = append(parts, styles.active.Render(text))
		case v == nil:
			parts = append(parts, styles.missing.Render(text))
		default:
			parts = append(parts, lipgloss.NewStyle().Padding(0, 1).Render(text))
		}
	}
	return strings.Join(parts, "")
}

func viewBody(a *substrate.Anchor, role string, w int) string {
	v := viewAt(a, role)
	var b strings.Builder
	if v == nil {
		why := "This anchor has no view at this commit."
		switch {
		case a.New:
			why = "Made by the work, so phoropter doesn't resolve it at any commit, including the as-built head where it now exists."
		case a.Target != nil:
			why = "It points at another plan's entry; plans don't change, so there's no commit to compare."
		case a.Note != "":
			why = a.Note + "."
		case role == "base":
			why = "An as-built's anchors are about code the work made, so they have no base."
		case role == "as-built head":
			why = "There's no as-built head for this plan, or it is the generated commit."
		}
		b.WriteString(styles.dim.Render(wrap(why, w)) + "\n")
		return b.String()
	}
	info := []string{v.File, v.Status}
	if v.Note != "" {
		info = append(info, ansi.Strip(v.Note))
	}
	b.WriteString(styles.dim.Render(wrap(strings.Join(info, " · "), w)) + "\n")
	if v.CodeFrom != "" {
		b.WriteString(styles.dim.Render("code repeated from "+v.CodeFrom) + "\n")
	}
	if v.Code == nil {
		b.WriteString("\n" + styles.dim.Render(wrap("The substrate gives no code for this view: it shows excerpts, and a quote it didn't find in a file leaves nothing to excerpt.", w)) + "\n")
		return b.String()
	}
	b.WriteString("\n" + renderCode(v.Code, w))
	return b.String()
}

func renderCode(ex *substrate.Excerpt, w int) string {
	var b strings.Builder
	for _, l := range ex.Lines {
		gutter := styles.gutter.Render(fmt.Sprintf("%5d │ ", l.N))
		text := ansi.Truncate(strings.ReplaceAll(l.Text, "\t", "    "), max(1, w-8), "…")
		if l.Marked {
			text = styles.marked.Render(text)
		}
		b.WriteString(gutter + text + "\n")
	}
	return b.String()
}

func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	if strings.HasSuffix(word, "y") {
		return fmt.Sprintf("%d %sies", n, strings.TrimSuffix(word, "y"))
	}
	return fmt.Sprintf("%d %ss", n, word)
}

// ---- a symbol or file, and every plan anchored there

type placeScreen struct {
	place  *substrate.Place
	cur    int
	scroll int
}

func (s *placeScreen) title() string { return s.place.Name }

func (s *placeScreen) help() string {
	return "j/k move · enter open · J/K scroll · r refresh · esc back"
}

func (s *placeScreen) refresh() ([]string, string) { return []string{s.place.URL}, s.place.URL }

func (s *placeScreen) targets() []menuItem {
	var out []menuItem
	for _, r := range s.place.AnchoredBy {
		out = append(out, menuItem{r.Kind, r.Title, r.URL})
	}
	if s.place.FileURL != "" {
		out = append(out, menuItem{"file", "the file page for " + s.place.File + " (guessed URL)", s.place.FileURL})
	}
	return out
}

func (s *placeScreen) key(m *Model, k string) tea.Cmd {
	t := s.targets()
	switch k {
	case "enter", "l", "right":
		if len(t) > 0 {
			return m.open(t[s.cur].url)
		}
	case "J", "ctrl+d", "pgdown":
		s.scroll += 5
	case "K", "ctrl+u", "pgup":
		s.scroll = max(0, s.scroll-5)
	case "esc", "backspace", "h", "left":
		m.back()
	default:
		if len(t) > 0 {
			s.cur = move(s.cur, len(t), k)
		}
	}
	return nil
}

func (s *placeScreen) view(m *Model, w, h int) string {
	lw := min(max(w*2/5, 30), w/2)
	var rows []string
	refs := s.place.AnchoredBy
	for i, t := range s.targets() {
		var text string
		if i < len(refs) {
			text = fmt.Sprintf("%-8s %s %s", refs[i].Plan, styles.kind(t.rel).Render(t.rel), t.label)
		} else {
			text = styles.dim.Render(t.label)
		}
		text = ansi.Truncate(text, lw, "…")
		if i == s.cur {
			text = styles.cursor.Render(ansi.Strip(text))
		}
		rows = append(rows, text)
	}
	rw := detailWidth(w)
	var b strings.Builder
	p := s.place
	b.WriteString(styles.title.Render(wrap(p.Name, rw)) + "\n")
	b.WriteString(styles.dim.Render(p.Kind+" · newest first · "+plural(len(refs), "entry")) + "\n\n")
	if s.cur < len(refs) {
		r := refs[s.cur]
		b.WriteString(field("where", r.Where, rw))
		info := r.Date + ", added in " + r.Added.Label
		if r.New {
			info += ", new"
		}
		b.WriteString(styles.dim.Render(info) + "\n\n")
	}
	if p.Code != nil {
		b.WriteString(styles.dim.Render(fmt.Sprintf("%s, lines %d–%d, at the generated commit", p.File, p.StartLine, p.EndLine)) + "\n\n")
		b.WriteString(renderCode(p.Code, rw))
	}
	detail := b.String()
	s.scroll = min(s.scroll, max(0, strings.Count(detail, "\n")+1-h))
	return split(listWindow(rows, s.cur, h), detail, w, h, s.scroll)
}
