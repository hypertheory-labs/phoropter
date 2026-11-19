package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hypertheory-labs/phoropter/internal/site"
)

// doctor reports what this machine has and lacks for the repository it's run
// in, and the command that would fix each thing. It runs none of them
// (grosgrain): Jeff hands piped install scripts to Claude to check, and this
// is the same help without that worry.
type doctorFacts struct {
	git         string // git's version, or "" when it isn't on PATH
	firstParty  string // site.FirstParty
	indexers    []site.IndexerInfo
	unavailable []site.Unavailable
	schemes     map[string]int // symbol anchors in this repository's plans, by scheme
}

// firstPartySchemes are the indexers phoropter ships, by scheme, so doctor
// can say where to get them when a binary outside a clone has none. One with
// an install command installs on its own (tulle); the rest need a clone.
var firstPartySchemes = map[string]struct{ lang, dir, toolchain, install string }{
	"scip-go": {"Go", "go", "Go 1.25 or newer (https://go.dev/dl)",
		"go install github.com/hypertheory-labs/phoropter/cmd/phoropter-index-scip-go@latest"},
	"scip-typescript": {"TypeScript", "typescript", "Node.js 24 or newer (https://nodejs.org), as its package.json says", ""},
}

// schemesInPlans counts the symbol anchors in the plans and as-builts at sha,
// by the scheme each one names: the languages this repository needs indexed.
func schemesInPlans(repo *site.Repo, sha string) map[string]int {
	schemes := map[string]int{}
	for _, f := range repo.Files(sha, "plans") {
		if !strings.HasSuffix(f, ".md") {
			continue
		}
		text, ok := repo.Show(sha, f)
		if !ok {
			continue
		}
		for _, e := range site.ParsePlan(text).Entries {
			for _, a := range e.Anchors {
				if site.IsScipSymbol(a.Where) {
					schemes[site.SchemeOf(a.Where)]++
				}
			}
		}
	}
	return schemes
}

// report is doctor's output, and whether anything this repository needs is
// missing.
func (d doctorFacts) report() (string, bool) {
	var b strings.Builder
	missing := false
	fix := func(format string, args ...any) {
		missing = true
		fmt.Fprintf(&b, "    fix: "+format+"\n", args...)
	}

	if d.git != "" {
		fmt.Fprintf(&b, "ok    %s\n", d.git)
	} else {
		b.WriteString("no    git: phoropter reads repositories through it\n")
		fix("install git (https://git-scm.com)")
	}

	byScheme := map[string]site.IndexerInfo{}
	for _, i := range d.indexers {
		byScheme[i.Scheme] = i
	}
	// Outside a clone, say so only when it costs this repository something:
	// a scheme of ours that it anchors and nothing here indexes.
	lacksOurs := false
	for s := range d.schemes {
		_, ours := firstPartySchemes[s]
		_, found := byScheme[s]
		lacksOurs = lacksOurs || (ours && !found)
	}
	if d.firstParty != "" {
		fmt.Fprintf(&b, "ok    phoropter's own indexers, in %s\n", d.firstParty)
	} else if lacksOurs {
		b.WriteString("no    phoropter's own indexers (Go, TypeScript): this binary isn't in a clone of phoropter\n")
		b.WriteString("    The Go one installs on its own with go install. The TypeScript one is in\n" +
			"    indexers/ in a clone, found from a binary built there, or from PHOROPTER_INDEXERS.\n")
	}
	for _, u := range d.unavailable {
		fmt.Fprintf(&b, "no    %s couldn't start: %v\n", shortPath(u.Where), u.Err)
	}

	var schemes []string
	for s := range d.schemes {
		schemes = append(schemes, s)
	}
	sort.Strings(schemes)
	if len(schemes) == 0 {
		b.WriteString("\nThis repository's plans anchor no symbols, so no indexer is needed yet.\n" +
			"File anchors with a quote work in any language.\n")
	} else {
		b.WriteString("\nWhat this repository's plans anchor:\n")
	}
	for _, s := range schemes {
		n := count(d.schemes[s], "symbol anchor")
		i, ok := byScheme[s]
		switch {
		case !ok:
			fmt.Fprintf(&b, "no    %s, %s: no indexer here for it\n", s, n)
			fp, ours := firstPartySchemes[s]
			if ours && d.firstParty == "" && fp.install != "" {
				fix("%s, which runs on %s", fp.install, fp.toolchain)
			} else if ours && d.firstParty == "" {
				fix("phoropter's %s indexer is in a clone of phoropter: git clone https://github.com/hypertheory-labs/phoropter, "+
					"then PHOROPTER_INDEXERS=<the clone>/indexers (or run a phoropter built in the clone)", fp.lang)
			} else if ours && d.couldntStart(fp.dir) {
				fix("phoropter's %s indexer runs on %s", fp.lang, fp.toolchain)
			} else {
				fix("an indexer named phoropter-index-%s on PATH; indexers/README.md in phoropter's repository says how one works", s)
			}
		case len(i.Needs) > 0:
			fmt.Fprintf(&b, "no    %s, %s: %s %s needs", s, n, i.Indexer, i.Version)
			for _, need := range i.Needs {
				fmt.Fprintf(&b, " %s", need.What)
			}
			b.WriteString("\n")
			for _, need := range i.Needs {
				fix("%s", need.Install)
			}
		default:
			fmt.Fprintf(&b, "ok    %s, %s: %s %s\n", s, n, i.Indexer, i.Version)
		}
	}
	if missing {
		b.WriteString("\nIndexes already committed in .phoropter/index/ still answer for their commits;\n" +
			"what's missing only matters for commits nobody has indexed yet.\n")
	}

	var others []string
	for _, i := range d.indexers {
		if _, used := d.schemes[i.Scheme]; !used {
			others = append(others, fmt.Sprintf("%s (%s)", i.Scheme, i.Indexer))
		}
	}
	if len(others) > 0 {
		fmt.Fprintf(&b, "\nAlso on this machine, unused here: %s.\n", strings.Join(others, ", "))
	}
	return b.String(), missing
}

// couldntStart: phoropter's own indexer in indexers/<dir> was found and
// couldn't describe itself, which is its toolchain missing.
func (d doctorFacts) couldntStart(dir string) bool {
	for _, u := range d.unavailable {
		if d.firstParty != "" && u.Where == filepath.Join(d.firstParty, dir) {
			return true
		}
	}
	return false
}

func gitVersion() string {
	if _, err := exec.LookPath("git"); err != nil {
		return ""
	}
	out, err := exec.Command("git", "--version").Output()
	if err != nil {
		return "git"
	}
	return strings.TrimSpace(string(out))
}

func shortPath(p string) string {
	if rel, err := filepath.Rel(filepath.Dir(site.FirstParty), p); site.FirstParty != "" && err == nil && !strings.HasPrefix(rel, "..") {
		return rel
	}
	return p
}
