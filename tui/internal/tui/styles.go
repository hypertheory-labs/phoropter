package tui

import "github.com/charmbracelet/lipgloss"

type styleSet struct {
	header, footer, title, dim, err, cursor, marked, gutter, active, missing, rule lipgloss.Style
	draftHeader, slot, slotEmpty, card, cardChosen                                 lipgloss.Style
}

// accent is a draft's color, used nowhere else. A palette number, so it
// follows the terminal's theme.
var accent = lipgloss.Color("13")

// quiet is for rules and borders: the theme's "bright black", which themes
// keep readable, rather than a fixed gray a theme's background can swallow.
var quiet = lipgloss.Color("8")

var styles = styleSet{
	header:  lipgloss.NewStyle().Bold(true).Padding(0, 1).Reverse(true),
	footer:  lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "240", Dark: "245"}),
	title:   lipgloss.NewStyle().Bold(true),
	dim:     lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "244", Dark: "243"}),
	err:     lipgloss.NewStyle().Foreground(lipgloss.Color("1")),
	cursor:  lipgloss.NewStyle().Reverse(true),
	marked:  lipgloss.NewStyle().Background(lipgloss.AdaptiveColor{Light: "229", Dark: "58"}),
	gutter:  lipgloss.NewStyle().Foreground(quiet),
	active:  lipgloss.NewStyle().Reverse(true).Bold(true).Padding(0, 1),
	missing: lipgloss.NewStyle().Foreground(quiet).Padding(0, 1),
	rule:    lipgloss.NewStyle().Foreground(quiet),

	draftHeader: lipgloss.NewStyle().Bold(true).Padding(0, 1).Background(accent).Foreground(lipgloss.Color("0")),
	slot:        lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(accent).Padding(0, 1),
	slotEmpty:   lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(quiet).Padding(0, 1),
	card:        lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(quiet).Padding(0, 1),
	cardChosen:  lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1),
}

var kindColors = map[string]string{
	"Decision": "4",
	"Decline":  "5",
	"Guess":    "3",
	"Changed":  "1",
	"Chose":    "6",
	"Answered": "2",
}

func (styleSet) kind(k string) lipgloss.Style {
	c, ok := kindColors[k]
	if !ok {
		c = "7"
	}
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(c))
}

var verdictColors = map[string]string{"held": "2", "disproved": "1", "partly disproved": "3", "not tested": "8"}

func (styleSet) verdict(v string) lipgloss.Style {
	c, ok := verdictColors[v]
	if !ok {
		c = "7"
	}
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(c))
}

var statusMarks = map[string]string{"found": "✓", "not-found": "✗", "no-quote": "·", "no-file": "∅", "no-symbol": "∅"}
