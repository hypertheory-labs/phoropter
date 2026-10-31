package site

import (
	"fmt"
	"regexp"
	"strings"
)

// A plan or as-built file in the lanyard format: frontmatter, a title, intro
// prose, then `## Kind: title` entries, each optionally starting with a list
// of anchors (`where:`, `where (new):`, `quote:`) and followed by prose.
//
// The structure is found by a line scanner that knows the format's few
// blocks (ATX headings and lists, outside code fences), not by a markdown
// library; goldmark only renders prose (organdy).

type Anchor struct {
	Where    string
	IsNew    bool
	Quote    string
	HasQuote bool
}

type Entry struct {
	Kind    string // Decision, Decline, Guess, Changed, Chose, Answered, or "" for a plain section
	Title   string // markdown, inline
	Slug    string
	Anchors []Anchor
	Body    string // markdown
	Source  string // the parenthesis in a leading **Because (…):** or **Answer (…):**, as written
	Verdict string // an Answered's opening words, when they're one of grommet's four
}

// Grommet: an answer opens with one of these. Longest first, so "partly
// disproved" isn't read as "disproved".
var verdicts = []string{"partly disproved", "not tested", "disproved", "held"}

type KV struct{ Key, Value string }

type PlanDoc struct {
	Frontmatter []KV
	Title       string
	Intro       string
	Entries     []*Entry
}

func (d *PlanDoc) FM(key string) (string, bool) {
	for _, kv := range d.Frontmatter {
		if kv.Key == key {
			return kv.Value, true
		}
	}
	return "", false
}

var (
	frontmatterRE = regexp.MustCompile(`(?s)^---\n(.*?)\n---\n?`)
	kvRE          = regexp.MustCompile(`^([\w-]+):\s*(.*)$`)
	headingRE     = regexp.MustCompile(`^ {0,3}(#{1,6})(?:[ \t]+(.*?))?(?:[ \t]+#+)?[ \t]*$`)
	entryRE       = regexp.MustCompile(`^(Decision|Decline|Guess|Changed|Chose|Answered):\s*(.*)$`)
	fenceRE       = regexp.MustCompile("^ {0,3}(```+|~~~+)")
	itemRE        = regexp.MustCompile(`^ {0,3}(?:[-*+]|\d{1,9}[.)])(?:[ \t]+(.*))?$`)
	anchorItemRE  = regexp.MustCompile(`(?s)^(where \(new\)|where|quote):\s*(.*)$`)
	sourceRE      = regexp.MustCompile(`^\*\*(?:Because|Answer) \((.*?)\):\*\*`)
	answerRE      = regexp.MustCompile(`(?s)^\*\*Answer \(.*?\):\*\*\s*(.*)`)
	letterRE      = regexp.MustCompile(`[a-zA-Z]`)
)

func ParsePlan(source string) *PlanDoc {
	doc := &PlanDoc{}
	rest := source
	if m := frontmatterRE.FindStringSubmatchIndex(source); m != nil {
		for _, line := range strings.Split(source[m[2]:m[3]], "\n") {
			if kv := kvRE.FindStringSubmatch(line); kv != nil {
				set := false
				for i := range doc.Frontmatter {
					if doc.Frontmatter[i].Key == kv[1] {
						doc.Frontmatter[i].Value, set = kv[2], true
					}
				}
				if !set {
					doc.Frontmatter = append(doc.Frontmatter, KV{kv[1], kv[2]})
				}
			}
		}
		rest = source[m[1]:]
	}

	lines := strings.Split(rest, "\n")
	slugs := map[string]bool{}
	var current *Entry
	var intro, body []string
	fence := ""
	flush := func() {
		if current != nil {
			current.Body = strings.Join(body, "\n")
		}
		body = nil
	}
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if fence != "" {
			if m := fenceRE.FindStringSubmatch(line); m != nil && m[1][0] == fence[0] && len(m[1]) >= len(fence) && strings.TrimSpace(line[strings.Index(line, m[1])+len(m[1]):]) == "" {
				fence = ""
			}
		} else if m := fenceRE.FindStringSubmatch(line); m != nil {
			fence = m[1]
		} else if h := headingRE.FindStringSubmatch(line); h != nil {
			text := strings.TrimSpace(h[2])
			if len(h[1]) == 1 && doc.Title == "" {
				doc.Title = text
				continue
			}
			if len(h[1]) == 2 {
				flush()
				title, kind := text, ""
				if m := entryRE.FindStringSubmatch(text); m != nil {
					kind, title = m[1], m[2]
				}
				current = &Entry{Kind: kind, Title: title, Slug: uniqueSlug(title, slugs)}
				doc.Entries = append(doc.Entries, current)
				// An anchor list is the first block after the heading.
				j := i + 1
				for j < len(lines) && strings.TrimSpace(lines[j]) == "" {
					j++
				}
				if j < len(lines) && itemRE.MatchString(lines[j]) {
					items, end := listItems(lines, j)
					if anchors, ok := parseAnchors(items); ok {
						current.Anchors = anchors
						i = end - 1
					}
				}
				continue
			}
		}
		if current == nil {
			intro = append(intro, line)
		} else {
			body = append(body, line)
		}
	}
	flush()
	doc.Intro = strings.TrimSpace(strings.Join(intro, "\n"))
	for _, e := range doc.Entries {
		e.Body = strings.TrimSpace(e.Body)
		if m := sourceRE.FindStringSubmatch(e.Body); m != nil {
			e.Source = m[1]
		}
		if e.Kind == "Answered" {
			after := ""
			if m := answerRE.FindStringSubmatch(e.Body); m != nil {
				after = m[1]
			}
			// "Held." is the same verdict as "held."; it's always given lowercase.
			lower := strings.ToLower(after)
			for _, v := range verdicts {
				if strings.HasPrefix(lower, v) && (len(after) == len(v) || !letterRE.MatchString(after[len(v):len(v)+1])) {
					e.Verdict = v
					break
				}
			}
		}
	}
	return doc
}

// listItems reads a list from its first item: each item's text without its
// marker, with continuation lines joined. The list goes on across blank lines
// while the next line is another item or indented. It returns the items and
// the line after the list.
func listItems(lines []string, start int) ([]string, int) {
	var items []string
	i := start
	for i < len(lines) {
		line := lines[i]
		if m := itemRE.FindStringSubmatch(line); m != nil {
			items = append(items, m[1])
			i++
			continue
		}
		if strings.TrimSpace(line) == "" {
			j := i
			for j < len(lines) && strings.TrimSpace(lines[j]) == "" {
				j++
			}
			if j < len(lines) && (itemRE.MatchString(lines[j]) || strings.HasPrefix(lines[j], "  ")) {
				i = j
				continue
			}
			break
		}
		if len(items) > 0 && (strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") || !itemRE.MatchString(line)) && !headingRE.MatchString(line) && !fenceRE.MatchString(line) {
			items[len(items)-1] += "\n" + strings.TrimSpace(line)
			i++
			continue
		}
		break
	}
	return items, i
}

// parseAnchors: an anchor list is one whose every item is `where:`,
// `where (new):`, or `quote:`. A `quote:` belongs to the `where` before it.
// Any other list is just prose.
func parseAnchors(items []string) ([]Anchor, bool) {
	var anchors []Anchor
	for _, item := range items {
		m := anchorItemRE.FindStringSubmatch(strings.TrimSpace(item))
		if m == nil {
			return nil, false
		}
		value := UnwrapCodeSpan(strings.TrimSpace(m[2]))
		if m[1] == "quote" {
			if len(anchors) == 0 || anchors[len(anchors)-1].HasQuote {
				return nil, false
			}
			anchors[len(anchors)-1].Quote, anchors[len(anchors)-1].HasQuote = value, true
		} else {
			anchors = append(anchors, Anchor{Where: value, IsNew: m[1] == "where (new)"})
		}
	}
	return anchors, true
}

// UnwrapCodeSpan reads a CommonMark code span: a run of backticks, the
// content, the same run. One leading and trailing space are stripped when both
// are present, which is how a span can start or end with a backtick.
func UnwrapCodeSpan(text string) string {
	n := 0
	for n < len(text) && text[n] == '`' {
		n++
	}
	if n == 0 || len(text) < 2*n {
		return text
	}
	run := strings.Repeat("`", n)
	if !strings.HasSuffix(text, run) || (len(text) > 2*n && text[len(text)-n-1] == '`') {
		return text
	}
	inner := text[n : len(text)-n]
	if len(inner) >= 2 && strings.HasPrefix(inner, " ") && strings.HasSuffix(inner, " ") && strings.TrimSpace(inner) != "" {
		return inner[1 : len(inner)-1]
	}
	return inner
}

var nonSlugRE = regexp.MustCompile(`[^a-z0-9]+`)

func Slugify(markdown string) string {
	s := strings.ToLower(strings.ReplaceAll(markdown, "`", ""))
	s = nonSlugRE.ReplaceAllString(s, "-")
	return strings.TrimSuffix(strings.TrimPrefix(s, "-"), "-")
}

func uniqueSlug(title string, taken map[string]bool) string {
	base := Slugify(title)
	if base == "" {
		base = "entry"
	}
	slug := base
	for n := 2; taken[slug]; n++ {
		slug = fmt.Sprintf("%s-%d", base, n)
	}
	taken[slug] = true
	return slug
}
