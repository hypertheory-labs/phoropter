// Package notes writes a draft's notes file by docs/annotating.md: answers to
// its prompts, and notes on its entries or on the whole draft. It never writes
// the draft. It reads the draft's source only for exact text (a prompt's
// heading, an entry's heading line) and positions; what a draft means comes
// from phoropter's page.
package notes

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// A Source is the draft's markdown, read for exact headings and line numbers.
type Source struct {
	Text    string
	Prompts []Heading // Question, Select, Affirm, in order
	Entries []Heading // the `## ` lines outside code fences, in order
}

type Heading struct {
	Line       int    // 1-based
	Raw        string // the whole line, as written
	Kind, Text string // for a prompt, the keyword and what follows it
}

var promptRE = regexp.MustCompile(`(?i)^#+\s*(Question|Select|Affirm):\s*(.*)$`)

func ParseSource(text string) *Source {
	s := &Source{Text: text}
	fenced := false
	for i, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "```") {
			fenced = !fenced
		}
		if fenced {
			continue
		}
		if m := promptRE.FindStringSubmatch(line); m != nil {
			s.Prompts = append(s.Prompts, Heading{Line: i + 1, Raw: line, Kind: canonical(m[1]), Text: strings.TrimSpace(m[2])})
		} else if strings.HasPrefix(line, "## ") {
			s.Entries = append(s.Entries, Heading{Line: i + 1, Raw: line})
		}
	}
	return s
}

func canonical(kind string) string {
	return strings.ToUpper(kind[:1]) + strings.ToLower(kind[1:])
}

// Prompt finds a prompt by kind and the text the page shows. The page renders
// inline markdown, so they're compared with backticks and emphasis removed.
func (s *Source) Prompt(kind, text string) (Heading, bool) {
	for _, p := range s.Prompts {
		if strings.EqualFold(p.Kind, kind) && plain(p.Text) == plain(text) {
			return p, true
		}
	}
	return Heading{}, false
}

func plain(s string) string {
	return strings.Join(strings.Fields(strings.NewReplacer("`", "", "*", "", "_", "").Replace(s)), " ")
}

// line is where a quote's first line is in the source, or 0.
func (s *Source) line(quote string) int {
	first := strings.SplitN(quote, "\n", 2)[0]
	for i, l := range strings.Split(s.Text, "\n") {
		if strings.Contains(l, first) {
			return i + 1
		}
	}
	return 0
}

// A File is a notes file: what comes before its first note, then the notes.
type File struct {
	Preamble []string
	Notes    []Note
}

// A Note is one `## ` heading and everything under it, kept as written.
type Note struct {
	Heading string   // without the `## `
	Body    []string // the lines after the heading
}

func Parse(text string) *File {
	f := &File{}
	fenced := false
	for _, line := range strings.Split(strings.TrimRight(text, "\n"), "\n") {
		if strings.HasPrefix(line, "```") {
			fenced = !fenced
		}
		if !fenced && strings.HasPrefix(line, "## ") {
			f.Notes = append(f.Notes, Note{Heading: strings.TrimSpace(line[3:])})
			continue
		}
		if len(f.Notes) == 0 {
			f.Preamble = append(f.Preamble, line)
		} else {
			n := &f.Notes[len(f.Notes)-1]
			n.Body = append(n.Body, line)
		}
	}
	return f
}

func (f *File) String() string {
	var b strings.Builder
	pre := strings.TrimRight(strings.Join(f.Preamble, "\n"), "\n")
	if pre != "" {
		b.WriteString(pre + "\n")
	}
	for _, n := range f.Notes {
		body := strings.TrimRight(strings.Join(n.Body, "\n"), "\n")
		b.WriteString("\n## " + n.Heading + "\n")
		if body != "" {
			b.WriteString(body + "\n")
		}
	}
	return b.String()
}

// quote is a note's blockquote, without the `> `, or "".
func (n Note) quote() string {
	var q []string
	for _, l := range n.Body {
		if strings.HasPrefix(l, ">") {
			q = append(q, strings.TrimPrefix(strings.TrimPrefix(l, ">"), " "))
		} else if strings.TrimSpace(l) != "" || len(q) > 0 {
			break
		}
	}
	return strings.Join(q, "\n")
}

// at is where a note belongs in the draft: its restated prompt's line, its
// quote's line, 0 for a note on the whole draft, or -1 when it can't be told.
func (n Note) at(src *Source) int {
	if m := promptRE.FindStringSubmatch("## " + n.Heading); m != nil {
		for _, p := range src.Prompts {
			if strings.EqualFold(p.Kind, m[1]) && p.Text == strings.TrimSpace(m[2]) {
				return p.Line
			}
		}
		return -1
	}
	q := n.quote()
	if q == "" {
		return 0
	}
	if l := src.line(q); l > 0 {
		return l
	}
	return -1
}

// put places a note in document order, after every note at or before line. A
// note it can't place stays where it was.
func (f *File) put(n Note, line int, src *Source) {
	idx := 0
	for i, m := range f.Notes {
		if at := m.at(src); at >= 0 && at <= line {
			idx = i + 1
		}
	}
	f.Notes = append(f.Notes[:idx], append([]Note{n}, f.Notes[idx:]...)...)
}

// Answer records an answer to a prompt, replacing any earlier one. lines are
// what goes under the restated heading: `**Answer:** …`, a Select's checked
// list, or `**Skipped.**`, then any `**Note:**`.
func (f *File) Answer(src *Source, p Heading, lines []string) {
	heading := p.Kind + ": " + p.Text
	for i, n := range f.Notes {
		if strings.EqualFold(n.Heading, heading) || n.Heading == heading {
			f.Notes = append(f.Notes[:i], f.Notes[i+1:]...)
			break
		}
	}
	f.put(Note{Heading: heading, Body: lines}, p.Line, src)
}

// Comment adds a note. With an entry heading, it quotes that line; without
// one, it's about the whole draft.
func (f *File) Comment(src *Source, entry *Heading, author, text string) {
	if entry == nil {
		f.put(Note{Heading: "— " + author, Body: []string{"", text}}, 0, src)
		return
	}
	n := Note{Heading: fmt.Sprintf("L%d — %s", entry.Line, author), Body: []string{"> " + entry.Raw, "", text}}
	f.put(n, entry.Line, src)
}

// SetAnnotates says which version of the draft the notes were written against.
func (f *File) SetAnnotates(draftName, draftText string) {
	sum := sha256.Sum256([]byte(strings.TrimSuffix(draftText, "\n")))
	line := fmt.Sprintf("<!-- annotates: %s sha256:%s -->", draftName, hex.EncodeToString(sum[:])[:12])
	for i, l := range f.Preamble {
		if strings.HasPrefix(l, "<!-- annotates:") {
			f.Preamble[i] = line
			return
		}
	}
	if len(f.Preamble) == 0 {
		f.Preamble = []string{"# Notes on " + draftName}
	}
	f.Preamble = append(f.Preamble[:1], append([]string{line}, f.Preamble[1:]...)...)
}

// ---- on disk

// A Draft is the draft on disk, checked against the page that showed it.
type Draft struct {
	Path, NotesPath string
	Source          *Source
	Notes           *File
	HasNotes        bool
}

var ErrMoved = errors.New("the draft on disk isn't the one on the page: it changed since the page was read (r reads it again)")

// Open finds the draft by walking up from dir to the repository's root, and
// checks that its sha256 is the one the page's header gave.
func Open(dir, file, pageSHA256 string) (*Draft, error) {
	root, err := repoRoot(dir)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(root, file)
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("no %s here: this TUI isn't running beside the draft's repository", file)
	}
	sum := sha256.Sum256(b)
	if hex.EncodeToString(sum[:]) != pageSHA256 {
		return nil, ErrMoved
	}
	d := &Draft{Path: path, NotesPath: strings.TrimSuffix(path, ".md") + ".notes.md", Source: ParseSource(string(b)), Notes: &File{}}
	if nb, err := os.ReadFile(d.NotesPath); err == nil {
		d.Notes, d.HasNotes = Parse(string(nb)), true
	}
	return d, nil
}

// Save writes the notes file. It never touches the draft.
func (d *Draft) Save() error {
	d.Notes.SetAnnotates(filepath.Base(d.Path), d.Source.Text)
	return os.WriteFile(d.NotesPath, []byte(d.Notes.String()), 0o644)
}

func repoRoot(dir string) (string, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("not inside a git repository, so there's no draft here to annotate")
		}
		dir = parent
	}
}

// Author is git's user.name, or $USER.
func Author() string {
	if out, err := exec.Command("git", "config", "user.name").Output(); err == nil {
		if name := strings.TrimSpace(string(out)); name != "" {
			return name
		}
	}
	if u := os.Getenv("USER"); u != "" {
		return u
	}
	return "someone"
}
