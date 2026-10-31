package site

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// The one draft plan, drafts/plan.draft.md, read from the working tree on
// every request (loden; tarlatan moved it into drafts/). Everything else on
// the site is read through git at a commit; a draft is uncommitted by nature,
// and says so.
//
// Its notes are drafts/plan.draft.notes.md, in the annotating agreement's
// format. A landed plan's notes are kept in drafts/ too, as
// drafts/<plan's file name>.notes.md (tarlatan): a person saying the plan
// matters to something now (organza).

const (
	DraftsDir  = "drafts"
	DraftFile  = "drafts/plan.draft.md"
	DraftNotes = "drafts/plan.draft.notes.md"
)

type Prompt struct {
	Kind     string // Question, Select, or Affirm
	Text     string // the heading after the keyword
	Body     string // markdown, without the options or the fallback
	Options  []string
	Fallback string // markdown, after **If unanswered:**
	Answer   *Annotation
}

type Annotation struct {
	Heading  string // as written: a location and author, or a restated prompt
	Quote    string // the source text it quotes, as written
	HasQuote bool
	Comment  string // markdown
	At       string // the slug of the entry holding the quote, or "intro"
	Found    bool   // false when the quote is nowhere in the source
}

type DraftNotesInfo struct {
	SHA256      string
	Annotates   string
	Current     bool
	Annotations []Annotation
}

type Draft struct {
	Plan    *Plan
	ReadAt  time.Time
	SHA256  string
	Prompts map[string][]*Prompt // by entry slug, or "intro"
	Notes   *DraftNotesInfo
}

func HasDraft(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, DraftFile))
	return err == nil
}

// PlanNotesFile is a landed plan's notes file, whether or not it exists.
func PlanNotesFile(p *Plan) string {
	return DraftsDir + "/" + strings.TrimSuffix(path.Base(p.File), ".md") + ".notes.md"
}

// ReadPlanNotes gives a landed plan's notes, each placed after the entry whose
// source holds its quote, and whether there are any.
func ReadPlanNotes(dir string, p *Plan) ([]Annotation, bool) {
	b, err := os.ReadFile(filepath.Join(dir, PlanNotesFile(p)))
	if err != nil {
		return nil, false
	}
	source, _ := os.ReadFile(filepath.Join(dir, p.File))
	var out []Annotation
	for _, a := range ParseNotes(string(b)) {
		out = append(out, placed(a, string(source), p))
	}
	return out, true
}

var restatedRE = regexp.MustCompile(`(?i)^(Question|Select|Affirm):\s*(.*)$`)
var annotatesRE = regexp.MustCompile(`annotates: \S+ sha256:([0-9a-f]+)`)

func ReadDraft(s *Site, dir string) (*Draft, error) {
	b, err := os.ReadFile(filepath.Join(dir, DraftFile))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	source := string(b)
	cx := s.Context
	doc := ParsePlan(source)

	prompts := map[string][]*Prompt{}
	intro, ip := SplitPrompts(doc.Intro)
	doc.Intro = intro
	if len(ip) > 0 {
		prompts["intro"] = ip
	}
	for _, e := range doc.Entries {
		body, ep := SplitPrompts(e.Body)
		e.Body = body
		if len(ep) > 0 {
			prompts[e.Slug] = ep
		}
	}

	base, hasBase := doc.FM("base")
	p := &Plan{Doc: Doc{File: DraftFile, Href: "draft/", Doc: doc}, Name: "draft"}
	p.Before, p.BeforeText = commitOf(cx.Repo, base, hasBase, "base")
	rv, _ := doc.FM("revisits")
	for _, q := range s.Plans {
		if q.File == rv {
			p.Revisits = q
		}
	}
	for _, e := range doc.Entries {
		p.Doc.Entries = append(p.Doc.Entries, &ResolvedEntry{Entry: e})
	}
	commits := []Commit{cx.Generated}
	if p.Before != nil && p.Before.SHA != cx.Generated.SHA {
		commits = append([]Commit{*p.Before}, commits...)
	}
	packages := PackagesOf(cx, cx.Docs)
	for k := range PackagesOf(cx, []*Doc{&p.Doc}) {
		packages[k] = true
	}
	ResolveEntries(cx, p, &p.Doc, commits, packages, false)

	d := &Draft{Plan: p, ReadAt: time.Now(), SHA256: sha(source), Prompts: prompts}
	if nb, err := os.ReadFile(filepath.Join(dir, DraftNotes)); err == nil {
		text := string(nb)
		var all []*Prompt
		for _, ps := range prompts {
			all = append(all, ps...)
		}
		var rest []Annotation
		for _, a := range ParseNotes(text) {
			var match *Prompt
			if m := restatedRE.FindStringSubmatch(a.Heading); m != nil {
				for _, pr := range all {
					if strings.EqualFold(pr.Kind, m[1]) && pr.Text == strings.TrimSpace(m[2]) {
						match = pr
						break
					}
				}
			}
			if match != nil {
				a := a
				match.Answer = &a
			} else {
				rest = append(rest, placed(a, source, p))
			}
		}
		notes := &DraftNotesInfo{SHA256: sha(text), Annotations: rest}
		if m := annotatesRE.FindStringSubmatch(text); m != nil {
			notes.Annotates = m[1]
			// annotations.nvim hashes the buffer's lines joined by newlines, which
			// drops the file's final newline; either form counts as this version.
			for _, v := range []string{source, strings.TrimSuffix(source, "\n")} {
				if strings.HasPrefix(sha(v), notes.Annotates) {
					notes.Current = true
				}
			}
		}
		d.Notes = notes
	}
	return d, nil
}

var (
	promptHeadingRE = regexp.MustCompile(`(?i)^(#{1,6})\s+(Question|Select|Affirm):\s*(.*)$`)
	anyHeadingRE    = regexp.MustCompile(`^(#{1,6})\s`)
	fallbackRE      = regexp.MustCompile(`^\*\*If unanswered:\*\*\s*(.*)$`)
	optionRE        = regexp.MustCompile(`^[-*]\s+(.*)$`)
)

// SplitPrompts takes each prompt out of a block of markdown. A prompt runs
// until the next heading of the same or a higher level, as the agreement says.
func SplitPrompts(md string) (string, []*Prompt) {
	lines := strings.Split(md, "\n")
	var prompts []*Prompt
	var rest []string
	for i := 0; i < len(lines); i++ {
		m := promptHeadingRE.FindStringSubmatch(lines[i])
		if m == nil {
			rest = append(rest, lines[i])
			continue
		}
		level := len(m[1])
		end := i + 1
		for end < len(lines) {
			if h := anyHeadingRE.FindStringSubmatch(lines[end]); h != nil && len(h[1]) <= level {
				break
			}
			end++
		}
		prompts = append(prompts, promptOf(m[2], strings.TrimSpace(m[3]), lines[i+1:end]))
		i = end - 1
	}
	return strings.TrimSpace(strings.Join(rest, "\n")), prompts
}

func promptOf(keyword, text string, lines []string) *Prompt {
	kind := strings.ToUpper(keyword[:1]) + strings.ToLower(keyword[1:])
	p := &Prompt{Kind: kind, Text: text}
	var body []string
	inOptions := false
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if f := fallbackRE.FindStringSubmatch(line); f != nil {
			para := []string{f[1]}
			for i+1 < len(lines) && strings.TrimSpace(lines[i+1]) != "" {
				i++
				para = append(para, lines[i])
			}
			p.Fallback = strings.TrimSpace(strings.Join(para, "\n"))
			continue
		}
		if o := optionRE.FindStringSubmatch(line); kind == "Select" && o != nil && (inOptions || len(p.Options) == 0) {
			inOptions = true
			p.Options = append(p.Options, o[1])
			continue
		}
		if inOptions && strings.TrimSpace(line) != "" {
			inOptions = false
		}
		body = append(body, line)
	}
	p.Body = strings.TrimSpace(strings.Join(body, "\n"))
	return p
}

var notesHeadingRE = regexp.MustCompile(`^## (.*)$`)

// ParseNotes reads a notes file: ## heading, an optional blockquote of the
// source, then the comment. The title and HTML comments before the first
// entry are skipped.
func ParseNotes(text string) []Annotation {
	var out []Annotation
	var heading string
	var lines []string
	have := false
	flush := func() {
		if !have {
			return
		}
		for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
			lines = lines[1:]
		}
		var quote []string
		for len(lines) > 0 && strings.HasPrefix(lines[0], ">") {
			q := strings.TrimPrefix(lines[0], ">")
			quote = append(quote, strings.TrimPrefix(q, " "))
			lines = lines[1:]
		}
		a := Annotation{Heading: heading, Comment: strings.TrimSpace(strings.Join(lines, "\n")), Found: true}
		if len(quote) > 0 {
			a.Quote, a.HasQuote = strings.Join(quote, "\n"), true
		}
		out = append(out, a)
	}
	for _, line := range strings.Split(text, "\n") {
		if m := notesHeadingRE.FindStringSubmatch(line); m != nil {
			flush()
			heading, lines, have = strings.TrimSpace(m[1]), nil, true
		} else if have {
			lines = append(lines, line)
		}
	}
	flush()
	return out
}

var fenceStartRE = regexp.MustCompile("^(```|~~~)")

// placed finds which entry's source holds the quote. A quote elided with …
// on its own line matches its first part, then its last part after it.
func placed(a Annotation, source string, p *Plan) Annotation {
	if !a.HasQuote {
		return a
	}
	parts := strings.Split(a.Quote, "\n…\n")
	first := parts[0]
	at := strings.Index(source, first)
	if at == -1 || (len(parts) > 1 && !strings.Contains(source[at+len(first):], parts[len(parts)-1])) {
		a.Found = false
		return a
	}
	// Entry headings are ## lines outside code fences, in the same order as
	// the parsed entries.
	offset, entry := 0, -1
	fenced := false
	for _, line := range strings.Split(source, "\n") {
		if offset > at {
			break
		}
		if fenceStartRE.MatchString(line) {
			fenced = !fenced
		} else if !fenced && strings.HasPrefix(line, "## ") {
			entry++
		}
		offset += len(line) + 1
	}
	a.At = "intro"
	if entry >= 0 && entry < len(p.Doc.Entries) {
		a.At = p.Doc.Entries[entry].Slug
	}
	return a
}

func sha(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}
