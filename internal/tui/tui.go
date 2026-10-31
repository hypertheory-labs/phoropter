// Package tui is a read-only terminal reader for a phoropter site: pick a
// plan, walk its entries, and flip each anchor between the commits phoropter
// shows it at, like the lenses of the instrument it's named after.
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"phoropter-tui/internal/substrate"
)

type screen interface {
	title() string
	help() string
	view(m *Model, w, h int) string
	key(m *Model, k string) tea.Cmd
	// refresh names the pages to read again, and where to land after.
	refresh() (forget []string, at string)
}

type Model struct {
	c        *substrate.Client
	ledger   *substrate.Ledger
	stack    []screen
	w, h     int
	status   string
	loading  bool
	showGaps bool
	showHelp bool
	scroll   int // for the overlays
}

func New(c *substrate.Client, ledger *substrate.Ledger) *Model {
	return &Model{c: c, ledger: ledger}
}

type loadedMsg struct {
	s       screen
	replace bool // a refresh: the screen takes the top's place
}
type errMsg struct{ err error }

func (m *Model) Init() tea.Cmd { return m.open("") }

// open loads the page at u and pushes the screen for it.
func (m *Model) open(u string) tea.Cmd {
	m.loading = true
	c := m.c
	return func() tea.Msg {
		s, err := load(c, u)
		if err != nil {
			return errMsg{err}
		}
		return loadedMsg{s, false}
	}
}

// reload reads the top screen's pages again and replaces it.
func (m *Model) reload() tea.Cmd {
	forget, at := m.top().refresh()
	for _, u := range forget {
		m.c.Forget(u)
	}
	m.loading = true
	c := m.c
	return func() tea.Msg {
		s, err := load(c, at)
		if err != nil {
			return errMsg{err}
		}
		return loadedMsg{s, true}
	}
}

// load decides what a page is from its main item's properties, not its URL.
func load(c *substrate.Client, u string) (screen, error) {
	p, err := c.Page(u)
	if err != nil {
		return nil, err
	}
	main := p.Main
	switch {
	case main.Get("entry") != nil || main.Get("title") != nil:
		d, err := c.Doc(u)
		if err != nil {
			return nil, err
		}
		plan, ab := d, (*substrate.Doc)(nil)
		if d.IsAsBuilt {
			if plan, err = c.Doc(d.PlanURL); err != nil {
				return nil, err
			}
			ab = d
		} else if d.AsBuiltURL != "" {
			if ab, err = c.Doc(d.AsBuiltURL); err != nil {
				return nil, err
			}
		}
		s := newDocScreen(plan, ab)
		s.selectURL(c.Resolve(u))
		return s, nil
	case main.Get("kind") != nil:
		pl, err := c.Place(u)
		if err != nil {
			return nil, err
		}
		return &placeScreen{place: pl}, nil
	default:
		h, err := c.Home()
		if err != nil {
			return nil, err
		}
		return &listScreen{home: h}, nil
	}
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
	case loadedMsg:
		m.loading, m.status = false, ""
		if msg.replace && len(m.stack) > 0 {
			m.stack[len(m.stack)-1] = msg.s
		} else {
			m.stack = append(m.stack, msg.s)
		}
	case errMsg:
		m.loading = false
		m.status = msg.err.Error()
	case tea.KeyMsg:
		k := msg.String()
		switch {
		case k == "ctrl+c" || k == "q":
			return m, tea.Quit
		case k == "g":
			m.showGaps, m.showHelp, m.scroll = !m.showGaps, false, 0
			return m, nil
		case k == "?":
			m.showHelp, m.showGaps, m.scroll = !m.showHelp, false, 0
			return m, nil
		case m.showGaps || m.showHelp:
			switch k {
			case "esc", "backspace":
				m.showGaps, m.showHelp = false, false
			case "j", "down":
				m.scroll++
			case "k", "up":
				m.scroll = max(0, m.scroll-1)
			}
			return m, nil
		}
		m.status = ""
		if len(m.stack) == 0 {
			return m, nil
		}
		if k == "r" {
			return m, m.reload()
		}
		if cmd := m.top().key(m, k); cmd != nil {
			return m, cmd
		}
	}
	return m, nil
}

func (m *Model) top() screen { return m.stack[len(m.stack)-1] }

// back pops a screen; it's the screen's job to close its own menus first.
func (m *Model) back() {
	if len(m.stack) > 1 {
		m.stack = m.stack[:len(m.stack)-1]
	}
}

func (m *Model) View() string {
	if m.w == 0 {
		return ""
	}
	if len(m.stack) == 0 {
		if m.status != "" {
			return "phoropter-tui: " + m.status + "\n\nq to quit"
		}
		return "loading " + m.c.Root.String() + " …"
	}
	var crumbs []string
	for _, s := range m.stack {
		crumbs = append(crumbs, s.title())
	}
	header := styles.header.Width(m.w).Render(ansi.Truncate(strings.Join(crumbs, " › "), m.w-2, "…"))

	gaps := len(m.ledger.Gaps())
	right := fmt.Sprintf("gaps %d (g) · ? help · q quit", gaps)
	left := m.top().help()
	if m.status != "" {
		left = styles.err.Render(m.status)
	} else if m.loading {
		left = "loading…"
	}
	pad := max(1, m.w-lipgloss.Width(left)-lipgloss.Width(right))
	footer := styles.footer.Render(ansi.Truncate(left+strings.Repeat(" ", pad)+right, m.w, "…"))

	h := m.h - lipgloss.Height(header) - lipgloss.Height(footer)
	var body string
	switch {
	case m.showGaps:
		body = clip(m.gapsView(m.w), m.w, h, m.scroll)
	case m.showHelp:
		body = clip(helpText, m.w, h, m.scroll)
	default:
		body = m.top().view(m, m.w, h)
	}
	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

func (m *Model) gapsView(w int) string {
	var b strings.Builder
	b.WriteString(styles.title.Render("Where this session read past the profile") + "\n")
	b.WriteString(styles.dim.Render("Each is a place the TUI parsed prose, leaned on markup the profile doesn't define, or went without. `phoropter-tui gaps` crawls the whole site.") + "\n\n")
	gaps := m.ledger.Gaps()
	if len(gaps) == 0 {
		b.WriteString("None yet.\n")
	}
	for _, g := range gaps {
		b.WriteString(styles.kind("Guess").Render(g.ID) + styles.dim.Render(fmt.Sprintf("  ×%d", g.Count)) + "\n")
		b.WriteString(wrap(g.Summary, w-2) + "\n")
		b.WriteString(styles.dim.Render(wrap("e.g. "+g.Example+"  "+g.Detail, w-2)) + "\n\n")
	}
	return b.String()
}

const helpText = `Keys

  plan list     j/k move · enter open (a draft, when there is one, is first)
  plan          j/k entry · tab next anchor · 1 2 3 or [ ] layer
                J/K scroll the right pane · enter links from here · esc back
  place         j/k move · enter open that plan at its entry · esc back
  anywhere      r read this page again · g gaps ledger · ? this help · q quit

Layers are the commits phoropter shows an anchor at:
  1 base           what the plan was deciding against
  2 as-built head  where the plan's own work landed
  3 generated      the commit this site was built from: "now"

A layer with nothing to show says why. That's the substrate's answer, not
an error.

A draft is the working tree, not a commit: its title says when it was read.
Its open questions (?) and notes (✎) follow the entry they belong to, and a
question shows what happens if nobody answers it.`

// clip lays text into a w×h box, starting at line offset.
func clip(text string, w, h, offset int) string {
	lines := strings.Split(text, "\n")
	offset = min(offset, max(0, len(lines)-h))
	lines = lines[offset:]
	if len(lines) > h {
		lines = lines[:h]
	}
	for i, l := range lines {
		lines[i] = ansi.Truncate(l, w, "…")
	}
	return lipgloss.NewStyle().Width(w).Height(h).Render(strings.Join(lines, "\n"))
}

func wrap(s string, w int) string {
	if w < 10 {
		w = 10
	}
	return lipgloss.NewStyle().Width(w).Render(s)
}
