package site

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Hit is one entry anchored where a question asks (chambray): in a file, or on
// lines a change touched.
type Hit struct {
	Plan  *Plan
	Doc   *Doc
	Entry *ResolvedEntry
	File  string // where it's anchored, at the site's generated commit
	Line  int    // its first line there, or 0 when it's the whole file
	Gone  bool   // its quote isn't found there: about the file, on no lines of it
}

// span is what an anchor covers at the generated commit: a file, and its
// lines, or nil lines for the whole file. Empty, not nil, lines are a quote
// that isn't found: the anchor is about the file, but on no line of it.
type span struct {
	file  string
	lines [][2]int
}

// anchoredSpan is where ref's anchor at place lands at the site's generated
// commit. A symbol covers its definition; a quote found in a file covers its
// matched lines; anything else covers the whole file.
func anchoredSpan(place *Place, ref PlaceRef) span {
	var gen *View
	for _, ra := range ref.Entry.Resolved {
		if ra.Place == place && ra.Anchor == ref.Anchor {
			for _, v := range ra.Views {
				if v.Role == "generated" {
					gen = v
				}
			}
		}
	}
	switch {
	case place.Kind == "symbol" && gen != nil && gen.File != "" && gen.StartLine > 0:
		return span{gen.File, [][2]int{{gen.StartLine, gen.EndLine}}}
	case place.Kind == "symbol" && place.Definition != nil:
		d := place.Definition
		return span{d.File, [][2]int{{d.StartLine, d.EndLine}}}
	case place.Kind == "symbol":
		return span{}
	case gen != nil && gen.Status == "found":
		s := span{file: place.Full}
		for _, m := range gen.At {
			if len(m) > 0 {
				s.lines = append(s.lines, [2]int{m[0], m[len(m)-1]})
			}
		}
		return s
	case gen != nil && gen.Status == "not-found":
		return span{file: place.Full, lines: [][2]int{}}
	}
	return span{file: place.Full}
}

// Affected lists the entries anchored where touches says yes, oldest plan
// first, each entry once. touches gets a file and the lines an anchor covers
// there (nil for the whole file).
func Affected(s *Site, touches func(file string, lines [][2]int) bool) []Hit {
	seen := map[*ResolvedEntry]bool{}
	var hits []Hit
	for _, place := range s.Places {
		for _, ref := range place.Refs {
			sp := anchoredSpan(place, ref)
			if sp.file == "" || seen[ref.Entry] || !touches(sp.file, sp.lines) {
				continue
			}
			seen[ref.Entry] = true
			h := Hit{Plan: ref.Plan, Doc: ref.Doc, Entry: ref.Entry, File: sp.file}
			if len(sp.lines) > 0 {
				h.Line = sp.lines[0][0]
			}
			h.Gone = sp.lines != nil && len(sp.lines) == 0
			hits = append(hits, h)
		}
	}
	order := map[*ResolvedEntry]int{}
	for i, p := range s.Plans {
		for _, d := range []*Doc{&p.Doc, p.AsBuilt} {
			if d == nil {
				continue
			}
			for j, e := range d.Entries {
				order[e] = i*100000 + j
				if d == p.AsBuilt {
					order[e] += 50000
				}
			}
		}
	}
	sort.SliceStable(hits, func(i, j int) bool { return order[hits[i].Entry] < order[hits[j].Entry] })
	return hits
}

// InFiles answers Affected for whole files: anything anchored in them.
func InFiles(files []string) func(string, [][2]int) bool {
	want := map[string]bool{}
	for _, f := range files {
		want[f] = true
	}
	return func(file string, _ [][2]int) bool { return want[file] }
}

// Changed is what a diff changed, in the old side's line numbers: per file,
// the line ranges, or nil when the whole file went (deleted or renamed).
type Changed map[string][][2]int

// Touches answers Affected for a change: an anchor's lines overlap a changed
// range, or the anchor or the change is the whole file.
func (c Changed) Touches(file string, lines [][2]int) bool {
	changed, ok := c[file]
	if !ok {
		return false
	}
	if changed == nil || lines == nil {
		return true
	}
	for _, a := range lines {
		for _, b := range changed {
			if a[0] <= b[1] && b[0] <= a[1] {
				return true
			}
		}
	}
	return false
}

var hunkRE = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+\d+(?:,\d+)? @@`)

// ParseDiff reads `git diff -U0 --no-renames` into what each file lost or
// changed, numbered as the old side has it. A pure insertion touches the
// lines on either side of it, since it lands between them.
func ParseDiff(diff string) Changed {
	c := Changed{}
	var file string
	for _, line := range strings.Split(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "--- "):
			file = ""
			if p := strings.TrimPrefix(line, "--- "); strings.HasPrefix(p, "a/") {
				file = unquotePath(p[2:])
			}
		case strings.HasPrefix(line, "+++ ") && file != "":
			if strings.TrimPrefix(line, "+++ ") == "/dev/null" {
				c[file] = nil
				file = ""
			} else if _, ok := c[file]; !ok {
				c[file] = [][2]int{}
			}
		case file != "":
			m := hunkRE.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			start, _ := strconv.Atoi(m[1])
			count := 1
			if m[2] != "" {
				count, _ = strconv.Atoi(m[2])
			}
			if count == 0 {
				c[file] = append(c[file], [2]int{start, start + 1})
			} else {
				c[file] = append(c[file], [2]int{start, start + count - 1})
			}
		}
	}
	return c
}

// unquotePath undoes git's quoting of a path with unusual characters.
func unquotePath(p string) string {
	if strings.HasPrefix(p, `"`) {
		if u, err := strconv.Unquote(p); err == nil {
			return u
		}
	}
	return p
}

// ChangedSince is what changed from where the working tree left ref (their
// merge base) to the working tree, and that merge base, which is the commit
// to read anchors at so both count lines the same way.
func ChangedSince(r *Repo, ref string) (Changed, string, error) {
	base, err := r.git("merge-base", ref, "HEAD")
	if err != nil {
		return nil, "", fmt.Errorf("%s and HEAD have no common commit", ref)
	}
	base = strings.TrimSpace(base)
	diff, err := r.git("diff", "-U0", "--no-renames", "--no-color", "--no-ext-diff", base)
	if err != nil {
		return nil, "", err
	}
	return ParseDiff(diff), base, nil
}

// LensLine is a hit as one line of a lens: the reference `Y` copies, and
// where it's anchored as the note.
func LensLine(h Hit) string {
	where := h.File
	if h.Line > 0 {
		where = fmt.Sprintf("%s:%d", h.File, h.Line)
	}
	if h.Gone {
		return fmt.Sprintf("- %s — %s, at `%s`, though its quote isn't found there now", wrapSpan(h.Doc.File+" › "+h.Entry.Title), strings.ToLower(h.Entry.Kind), where)
	}
	return fmt.Sprintf("- %s — %s, at `%s`", wrapSpan(h.Doc.File+" › "+h.Entry.Title), strings.ToLower(h.Entry.Kind), where)
}

// wrapSpan puts s in a code span that its own backticks can't close.
func wrapSpan(s string) string {
	longest, run := 0, 0
	for _, r := range s {
		if r == '`' {
			run++
			longest = max(longest, run)
		} else {
			run = 0
		}
	}
	fence := strings.Repeat("`", longest+1)
	if longest > 0 {
		return fence + " " + s + " " + fence
	}
	return fence + s + fence
}
