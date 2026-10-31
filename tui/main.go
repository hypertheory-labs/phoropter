// phoropter-tui reads a phoropter site in the terminal.
//
//	phoropter-tui [site]        browse; site is a URL or a built site's directory
//	phoropter-tui gaps [site]   crawl the site and print where it read past the profile
//
// The site defaults to http://localhost:4173/, what phoropter's npm run dev serves.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"phoropter-tui/internal/substrate"
	"phoropter-tui/internal/tui"
)

const defaultSite = "http://localhost:4173/"

func main() {
	args := os.Args[1:]
	gaps := len(args) > 0 && args[0] == "gaps"
	if gaps {
		args = args[1:]
	}
	site := defaultSite
	if len(args) > 0 {
		site = args[0]
	}
	ledger := substrate.NewLedger()
	c, err := substrate.NewClient(site, ledger)
	if err != nil {
		fail(err)
	}
	if gaps {
		s, err := c.Crawl()
		if err != nil {
			fail(err)
		}
		fmt.Printf("%s at %s: %d plans, %d places\n\n", s.Home.Repository, s.Home.Commit, len(s.Home.Plans), len(s.Places))
		fmt.Print(ledger.Report())
		return
	}
	if _, err := tea.NewProgram(tui.New(c, ledger), tea.WithAltScreen()).Run(); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "phoropter-tui:", err)
	os.Exit(1)
}
