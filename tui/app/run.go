package app

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/hypertheory-labs/phoropter/tui/substrate"
)

// Run is `phoropter ui` (organdy): the terminal reader, given a site's address
// or a built site's directory.
//
//	phoropter ui [site]        browse
//	phoropter ui gaps [site]   crawl the site and print where it read past the profile
//
// The TUI reads only what phoropter serves. It reaches phoropter over HTTP, or
// a built site on disk, and imports none of phoropter's own packages; a test
// keeps it that way.
func Run(args []string, site string) error {
	gaps := len(args) > 0 && args[0] == "gaps"
	if gaps {
		args = args[1:]
	}
	if len(args) > 0 {
		site = args[0]
	}
	ledger := substrate.NewLedger()
	c, err := substrate.NewClient(site, ledger)
	if err != nil {
		return err
	}
	if gaps {
		s, err := c.Crawl()
		if err != nil {
			return err
		}
		fmt.Printf("%s at %s: %d plans, %d places\n\n", s.Home.Repository, s.Home.Commit, len(s.Home.Plans), len(s.Places))
		fmt.Print(ledger.Report())
		return nil
	}
	_, err = tea.NewProgram(New(c, ledger), tea.WithAltScreen()).Run()
	return err
}
