package tui

import "github.com/charmbracelet/lipgloss"

type styleSet struct {
	header, footer, title, dim, err, cursor, marked, gutter, active, missing, rule lipgloss.Style
}

var styles = styleSet{
	header:  lipgloss.NewStyle().Bold(true).Padding(0, 1).Background(lipgloss.AdaptiveColor{Light: "254", Dark: "236"}),
	footer:  lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "240", Dark: "245"}),
	title:   lipgloss.NewStyle().Bold(true),
	dim:     lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "244", Dark: "243"}),
	err:     lipgloss.NewStyle().Foreground(lipgloss.Color("1")),
	cursor:  lipgloss.NewStyle().Reverse(true),
	marked:  lipgloss.NewStyle().Background(lipgloss.AdaptiveColor{Light: "229", Dark: "58"}),
	gutter:  lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "248", Dark: "240"}),
	active:  lipgloss.NewStyle().Reverse(true).Bold(true).Padding(0, 1),
	missing: lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "250", Dark: "238"}).Padding(0, 1),
	rule:    lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "250", Dark: "238"}),
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
