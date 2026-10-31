package site

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Shows an anchor's code at one commit. Symbols go through the index;
// everything else is a quote searched for in the anchor's own file only.
// Nothing here judges a difference between two commits.

type TargetKind string

const (
	TSymbol    TargetKind = "symbol"
	TPlan      TargetKind = "plan"
	TFile      TargetKind = "file"
	TCrossRepo TargetKind = "cross-repo"
)

type Target struct {
	Kind     TargetKind
	Symbol   string // version-less
	File     string
	Title    string // a plan's entry, when HasTitle
	HasTitle bool
	Path     []string
	Repo     string
	Rest     string
}

const (
	contextLines = 3
	declaration  = 2*contextLines + 1 // as many lines as a found quote's view, at most
)

var crossRE = regexp.MustCompile(`^([\w.-]+): (.+)$`)

// TargetOf: plans is the plan and as-built files on this site; an anchor into
// one of them points at the plan, or at an entry by its title, not at a file.
func TargetOf(a Anchor, repoName string, plans map[string]bool) Target {
	where := a.Where
	if m := crossRE.FindStringSubmatch(a.Where); m != nil {
		if m[1] != repoName {
			return Target{Kind: TCrossRepo, Repo: m[1], Rest: m[2]}
		}
		where = m[2]
	}
	if IsScipSymbol(where) {
		return Target{Kind: TSymbol, Symbol: WithoutVersion(where)}
	}
	parts := strings.Split(where, " › ")
	file := strings.TrimSpace(parts[0])
	if plans[file] {
		t := Target{Kind: TPlan, File: file}
		if len(parts) > 1 {
			t.Title, t.HasTitle = strings.Join(parts[1:], " › "), true
		}
		return t
	}
	t := Target{Kind: TFile, File: file}
	for _, p := range parts[1:] {
		t.Path = append(t.Path, strings.TrimSpace(p))
	}
	return t
}

type Hunk struct {
	Start  int // 1-based line number of Lines[0]
	Lines  []string
	Marked []int // line numbers the quote covers
}

type View struct {
	SHA, Short, Role string // what this commit is to the entry: base, as-built head, generated
	Status           string // found, not-found, no-quote, no-file, no-symbol, no-index
	File             string
	StartLine        int // the symbol's range, for symbol anchors
	EndLine          int
	Matches          int
	At               [][]int // the lines each match covers, in order
	Hunks            []Hunk
	Note             string // plain words for the reader
	SameAs           string // short sha of the as-built head this view matches exactly; its code is not repeated
}

func newView(c Commit) *View {
	return &View{SHA: c.SHA, Short: c.Short, Role: c.Role, Status: "no-quote"}
}

// Collapse: when the generated view matches the as-built head's exactly, it
// says so and drops its code. A fact about two commits, not a judgment.
func Collapse(views []*View) []*View {
	var head *View
	for _, v := range views {
		if v.Role == "as-built head" {
			head = v
			break
		}
	}
	if head == nil || len(views) == 0 {
		return views
	}
	last := views[len(views)-1]
	if last == head || last.Role != "generated" {
		return views
	}
	same := func(v *View) string {
		return fmt.Sprintf("%s|%s|%d|%d|%d|%v", v.Status, v.File, v.StartLine, v.EndLine, v.Matches, v.Hunks)
	}
	if same(head) != same(last) {
		return views
	}
	last.SameAs = head.Short
	last.Note = fmt.Sprintf("same as at `%s`", head.Short)
	last.Hunks = nil
	return views
}

func ResolveAt(repo *Repo, t Target, quote string, hasQuote bool, c Commit, index *Lookup) *View {
	view := newView(c)
	needle := strings.TrimSpace(quote)
	if !hasQuote {
		needle = ""
	}

	if t.Kind == TSymbol {
		var def Definition
		ok := false
		if index != nil {
			def, ok = index.Definitions[t.Symbol]
		}
		if !ok && index != nil {
			if failed, f := index.Failed[PackageKey(t.Symbol)]; f {
				// A missing or failing indexer is a status, not a failed build (batik).
				view.Status = "no-index"
				view.Note = fmt.Sprintf("not indexed at %s: %s", c.Short, failed)
				return view
			}
		}
		if !ok {
			view.Status = "no-symbol"
			view.Note = fmt.Sprintf("symbol not in the index at %s", c.Short)
			return view
		}
		text, _ := repo.Show(c.SHA, def.File)
		lines := strings.Split(text, "\n")
		view.File, view.StartLine, view.EndLine = def.File, def.StartLine, def.EndLine
		rng := fmt.Sprintf("lines %d–%d", def.StartLine, def.EndLine)
		var found [][]int
		if needle != "" {
			for _, m := range findQuote(slice(lines, def.StartLine-1, def.EndLine), needle) {
				var shifted []int
				for _, n := range m {
					shifted = append(shifted, n+def.StartLine-1)
				}
				found = append(found, shifted)
			}
		}
		view.Matches, view.At = len(found), found
		if len(found) > 0 {
			view.Status = "found"
			for _, m := range found {
				view.Hunks = append(view.Hunks, around(lines, m, def.StartLine, def.EndLine))
			}
			view.Note = fmt.Sprintf("%s; quote at %s", rng, describeLines(flatten(found)))
			return view
		}
		// No quote to centre on, so show the declaration: the symbol is still
		// here, even if the quote isn't. The whole range is on the symbol page.
		end := min(def.EndLine, def.StartLine+declaration-1)
		view.Hunks = []Hunk{{Start: def.StartLine, Lines: slice(lines, def.StartLine-1, end), Marked: []int{}}}
		if needle != "" {
			view.Status = "not-found"
			view.Note = fmt.Sprintf("%s; quote not found at `%s`, so this is the declaration", rng, c.Short)
		} else {
			view.Note = rng + "; no quote, so this is the declaration"
		}
		return view
	}

	view.File = t.File
	text, ok := repo.Show(c.SHA, t.File)
	if !ok {
		view.Status = "no-file"
		view.Note = fmt.Sprintf("`%s` doesn't exist at `%s`", t.File, c.Short)
		return view
	}
	if needle == "" {
		view.Note = "no quote, so nothing to search for"
		return view
	}
	lines := strings.Split(text, "\n")
	found := findQuote(lines, needle)
	view.Matches, view.At = len(found), found
	if len(found) == 0 {
		view.Status = "not-found"
		view.Note = fmt.Sprintf("quote not found at `%s`", c.Short)
		return view
	}
	view.Status = "found"
	for _, m := range found {
		view.Hunks = append(view.Hunks, around(lines, m, 1, len(lines)))
	}
	where := describeLines(flatten(found))
	if len(found) == 1 {
		view.Note = "quote at " + where
	} else {
		view.Note = fmt.Sprintf("quote matches %d times, at %s", len(found), where)
	}
	return view
}

var entryHeadingRE = regexp.MustCompile(`^## (?:\w+: )?`)

// ResolveInEntry: a quote inside one entry of a plan or as-built, at one
// commit. Both are frozen once their work lands (awl), so one commit is
// enough. The search stays inside the entry, from its heading to the next.
func ResolveInEntry(repo *Repo, file, title, quote string, c Commit) *View {
	view := newView(c)
	view.Status, view.File = "not-found", file
	text, _ := repo.Show(c.SHA, file)
	lines := strings.Split(text, "\n")
	heading := -1
	for i, l := range lines {
		if strings.HasPrefix(l, "## ") && entryHeadingRE.ReplaceAllString(l, "") == title {
			heading = i
			break
		}
	}
	if heading == -1 {
		view.Note = fmt.Sprintf("no entry titled this at `%s`", c.Short)
		return view
	}
	end := len(lines)
	for i := heading + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "## ") {
			end = i
			break
		}
	}
	for end > heading+1 && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	var found [][]int
	for _, m := range findQuote(lines[heading:end], strings.TrimSpace(quote)) {
		var shifted []int
		for _, n := range m {
			shifted = append(shifted, n+heading)
		}
		found = append(found, shifted)
	}
	view.Matches, view.At = len(found), found
	if len(found) == 0 {
		view.Note = fmt.Sprintf("quote not found in this entry at `%s`", c.Short)
		return view
	}
	view.Status = "found"
	view.Hunks = []Hunk{{Start: heading + 1, Lines: lines[heading:end], Marked: flatten(found)}}
	view.Note = "quote at " + describeLines(flatten(found))
	return view
}

// around is a match with context lines either side, kept within lines lo..hi
// (1-based).
func around(lines []string, match []int, lo, hi int) Hunk {
	start := max(lo, match[0]-contextLines)
	end := min(hi, match[len(match)-1]+contextLines)
	return Hunk{Start: start, Lines: slice(lines, start-1, end), Marked: match}
}

// findQuote: each match is the list of 1-based line numbers (relative to
// lines) that it spans. A quote may span lines.
func findQuote(lines []string, needle string) [][]int {
	text := strings.Join(lines, "\n")
	starts := []int{0}
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' {
			starts = append(starts, i+1)
		}
	}
	lineOf := func(offset int) int {
		return sort.Search(len(starts), func(i int) bool { return starts[i] > offset })
	}
	var matches [][]int
	for from := 0; from <= len(text); {
		at := strings.Index(text[from:], needle)
		if at < 0 || needle == "" {
			break
		}
		at += from
		first, last := lineOf(at), lineOf(at+len(needle)-1)
		var m []int
		for n := first; n <= last; n++ {
			m = append(m, n)
		}
		matches = append(matches, m)
		from = at + 1
	}
	return matches
}

func describeLines(lines []int) string {
	seen := map[int]bool{}
	var unique []int
	for _, n := range lines {
		if !seen[n] {
			seen[n] = true
			unique = append(unique, n)
		}
	}
	sort.Ints(unique)
	parts := make([]string, len(unique))
	for i, n := range unique {
		parts[i] = fmt.Sprint(n)
	}
	if len(unique) == 1 {
		return "line " + parts[0]
	}
	return "lines " + strings.Join(parts, ", ")
}

func flatten(m [][]int) []int {
	var out []int
	for _, x := range m {
		out = append(out, x...)
	}
	return out
}

// slice is JavaScript's Array.slice on lines: clamped, never panicking.
func slice(lines []string, from, to int) []string {
	from = max(0, min(from, len(lines)))
	to = max(from, min(to, len(lines)))
	return lines[from:to]
}
