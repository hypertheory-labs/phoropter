package site

import (
	"regexp"
	"strings"
)

// SCIP symbols in plans are stored without the package version (lanyard), so
// both sides are compared with the version replaced by ".". Indexing itself is
// done by programs in indexers/ (seersucker); this is the grammar they share.

type Definition struct {
	Symbol    string // version-less, as stored
	File      string // repo-relative
	StartLine int    // 1-based, inclusive
	EndLine   int
}

var symbolRE = regexp.MustCompile(`^[a-z][\w.-]* \S+ \S+ \S+ \S.*[#.):!/\]]$`)

// IsScipSymbol: scheme, manager, package, version, then descriptors ending in
// one of SCIP's suffixes. Any scheme, so `roslyn nuget …` is a symbol too. A
// `›` path is a file anchor, never a symbol.
func IsScipSymbol(where string) bool {
	return !strings.Contains(where, "›") && symbolRE.MatchString(where)
}

func SchemeOf(symbol string) string { return strings.SplitN(symbol, " ", 2)[0] }

// PackageKey is which indexer's package a symbol belongs to: its scheme and
// package name.
func PackageKey(symbol string) string {
	p := strings.Split(symbol, " ")
	if len(p) < 3 {
		return symbol
	}
	return p[0] + " " + p[2]
}

func WithoutVersion(symbol string) string {
	p := strings.Split(symbol, " ")
	if len(p) < 5 {
		return symbol
	}
	p[3] = "."
	return strings.Join(p, " ")
}
