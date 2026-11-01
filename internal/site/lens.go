package site

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Lenses (percale): a reader's own subsets of plans, in drafts/lenses/<name>.md.
// A lens is a title, a line or two of intro, and one reference per line in the
// syntax a plan's where: uses, each optionally followed by a note to oneself.
// The dev server shows them; a built site never does, since they're someone's
// way of reading the testimony, not part of it.

const LensDir = "drafts/lenses"

type Lens struct {
	Name       string // the file's name, without .md
	File       string // repo-relative
	Title      string
	Intro      string
	Repository string // the first commit's sha, as written
	Items      []LensItem
}

type LensItem struct {
	Ref, Note string
	Plan      *Plan
	Doc       *Doc
	Entry     *ResolvedEntry // nil for a whole plan
	Problem   string         // why it didn't resolve
}

var lensNoteSep = regexp.MustCompile(`^\s*(—|–|-|:)?\s*`)

// codeSpan splits a leading code span from the rest of a line: its run of
// backticks closes at the next run of the same length, as in markdown.
func codeSpan(s string) (inner, rest string, ok bool) {
	n := 0
	for n < len(s) && s[n] == '`' {
		n++
	}
	if n == 0 {
		return "", s, false
	}
	for i := n; i < len(s); {
		if s[i] != '`' {
			i++
			continue
		}
		j := i
		for j < len(s) && s[j] == '`' {
			j++
		}
		if j-i == n {
			return UnwrapCodeSpan(s[:j]), s[j:], true
		}
		i = j
	}
	return "", s, false
}

// ReadLenses reads every lens in the repository's drafts/lenses/, by name.
func ReadLenses(s *Site, repoDir string) []*Lens {
	files, _ := filepath.Glob(filepath.Join(repoDir, LensDir, "*.md"))
	sort.Strings(files)
	var out []*Lens
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		l := ParseLens(string(b))
		l.Name = strings.TrimSuffix(filepath.Base(f), ".md")
		l.File = LensDir + "/" + filepath.Base(f)
		if l.Title == "" {
			l.Title = l.Name
		}
		ResolveLens(s, l)
		out = append(out, l)
	}
	return out
}

func ParseLens(source string) *Lens {
	l := &Lens{}
	if m := frontmatterRE.FindStringSubmatchIndex(source); m != nil {
		for _, line := range strings.Split(source[m[2]:m[3]], "\n") {
			if k, v, ok := strings.Cut(line, ":"); ok && strings.TrimSpace(k) == "repository" {
				l.Repository = strings.TrimSpace(v)
			}
		}
		source = source[m[1]:]
	}
	var intro []string
	for _, line := range strings.Split(source, "\n") {
		t := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(t, "# ") && l.Title == "":
			l.Title = strings.TrimSpace(t[2:])
		case strings.HasPrefix(t, "- ") || strings.HasPrefix(t, "* "):
			if ref, rest, ok := codeSpan(strings.TrimSpace(t[2:])); ok {
				l.Items = append(l.Items, LensItem{Ref: strings.TrimSpace(ref), Note: strings.TrimSpace(lensNoteSep.ReplaceAllString(rest, ""))})
			} else {
				ref, note, _ := strings.Cut(strings.TrimSpace(t[2:]), " — ")
				l.Items = append(l.Items, LensItem{Ref: strings.TrimSpace(ref), Note: strings.TrimSpace(note)})
			}
		case len(l.Items) == 0 && l.Title != "":
			intro = append(intro, line)
		}
	}
	l.Intro = strings.TrimSpace(strings.Join(intro, "\n"))
	return l
}

// ResolveLens finds each item's plan, and its entry when it names one.
func ResolveLens(s *Site, l *Lens) {
	for i := range l.Items {
		it := &l.Items[i]
		file, title, hasTitle := strings.Cut(it.Ref, " › ")
		file, title = strings.TrimSpace(file), strings.TrimSpace(title)
		for _, p := range s.Plans {
			switch file {
			case p.File:
				it.Plan, it.Doc = p, &p.Doc
			default:
				if p.AsBuilt != nil && p.AsBuilt.File == file {
					it.Plan, it.Doc = p, p.AsBuilt
				}
			}
			if it.Doc != nil {
				break
			}
		}
		if it.Doc == nil {
			it.Problem = "not a plan on this site"
			continue
		}
		if !hasTitle {
			continue
		}
		// The title as the heading has it, or with the entry's kind in front,
		// the way pages show it: an agent wrote "Decline: …" for six of 25
		// lines before this (percale's as-built).
		for _, e := range it.Doc.Entries {
			if e.Title == title || (e.Kind != "" && title == e.Kind+": "+e.Title) {
				it.Entry = e
				break
			}
		}
		if it.Entry == nil {
			it.Problem = "no entry titled this in " + it.Plan.Name
		}
	}
}

// FirstCommit is the repository's root commit, which every clone shares, so a
// lens says which repository it's about. With several roots, the oldest.
func FirstCommit(r *Repo) string {
	out, err := r.git("rev-list", "--max-parents=0", "HEAD")
	if err != nil {
		return ""
	}
	lines := strings.Fields(out)
	if len(lines) == 0 {
		return ""
	}
	return lines[len(lines)-1]
}

// openGuess: unanswered, or answered "not tested", so later work might be its test.
func openGuess(e *ResolvedEntry) bool {
	n := len(e.AnsweredBy)
	return e.Kind == "Guess" && (n == 0 || e.AnsweredBy[n-1].Entry.Verdict == "not tested")
}

func LensesPage(s *Site, lenses []*Lens, first string) Page {
	here := "lenses/"
	var items []string
	for _, l := range lenses {
		items = append(items, fmt.Sprintf(`<li itemprop="lens" itemscope><a rel="item" itemprop="url" href="%s/"><span itemprop="title">%s</span></a> <span class="meta"><code itemprop="file">%s</code>, <span itemprop="itemCount">%d</span> items</span></li>`, esc(l.Name), inline(l.Title), esc(l.File), len(l.Items)))
	}
	list := `<p class="note">No lenses yet. A lens is a file in <code>drafts/lenses/</code>: a title, then one reference per line, <code>plans/&lt;file&gt;.md › &lt;entry title&gt;</code>.</p>`
	if len(items) > 0 {
		list = "<ul class=\"lenses\">\n" + strings.Join(items, "\n") + "\n</ul>"
	}
	return layout(s, here, "Lenses", fmt.Sprintf(`
<nav><a href="%s">%s at %s</a></nav>
<article class="lenses" itemscope>
<h1>Lenses</h1>
<p class="draft-label">A reader's own subsets of plans, from <code>drafts/lenses/</code>. Not testimony.</p>
<p class="meta">This repository's first commit: <data itemprop="firstCommit" value="%s">%s</data></p>
%s
</article>`, Rel(here, ""), esc(s.RepoName), esc(s.Generated.Short), first, esc(short(first)), list), "")
}

func LensPage(s *Site, l *Lens, first string) Page {
	here := "lenses/" + l.Name + "/"
	var items []string
	for _, it := range l.Items {
		var what string
		switch {
		case it.Problem != "":
			what = fmt.Sprintf(`<span class="note" itemprop="problem">%s</span>`, esc(it.Problem))
		case it.Entry != nil:
			status := ""
			if it.Entry.Kind == "Answered" && it.Entry.Verdict != "" {
				status = ` <span class="verdict" itemprop="status">` + esc(it.Entry.Verdict) + `</span>`
			}
			if it.Entry.Kind == "Guess" {
				if openGuess(it.Entry) {
					status = ` <span class="verdict" itemprop="status">open</span>`
				} else {
					status = ` <span class="verdict" itemprop="status">` + esc(it.Entry.AnsweredBy[len(it.Entry.AnsweredBy)-1].Entry.Verdict) + `</span>`
				}
			}
			what = fmt.Sprintf(`<a rel="lens-item" itemprop="entry" href="%s#%s"><span class="kind kind-%s" itemprop="kind">%s</span>: <span itemprop="title">%s</span></a> <span class="meta">in <span itemprop="plan">%s</span></span>%s`, Rel(here, it.Doc.Href), it.Entry.Slug, strings.ToLower(it.Entry.Kind), esc(it.Entry.Kind), inline(it.Entry.Title), docName(EntryRef{Plan: it.Plan, Doc: it.Doc}), status)
		default:
			what = fmt.Sprintf(`<a rel="lens-item" itemprop="entry" href="%s"><span itemprop="kind">Plan</span>: <span itemprop="title">%s</span></a> <span class="meta">in <span itemprop="plan">%s</span></span>`, Rel(here, it.Doc.Href), inline(it.Doc.Doc.Title), docName(EntryRef{Plan: it.Plan, Doc: it.Doc}))
		}
		note := ""
		if it.Note != "" {
			note = fmt.Sprintf(`<p class="note-text" itemprop="note">%s</p>`, inline(it.Note))
		}
		items = append(items, fmt.Sprintf(`<li itemprop="lensItem" itemscope>%s
<p class="meta"><code itemprop="reference">%s</code></p>%s</li>`, what, esc(it.Ref), note))
	}
	intro := ""
	if l.Intro != "" {
		intro = fmt.Sprintf(`<div class="intro" itemprop="intro">%s</div>`, block(l.Intro))
	}
	repo := `<dd class="note">not given</dd>`
	if l.Repository != "" {
		repo = fmt.Sprintf(`<dd><code itemprop="repository">%s</code></dd>`, esc(l.Repository))
		if first != "" && !strings.HasPrefix(first, l.Repository) && !strings.HasPrefix(l.Repository, first) {
			repo = fmt.Sprintf(`<dd><code itemprop="repository">%s</code> <span class="note" itemprop="repositoryNote">another repository: this one's first commit is %s</span></dd>`, esc(l.Repository), esc(short(first)))
		}
	}
	return layout(s, here, l.Title, fmt.Sprintf(`
<nav><a href="%s">%s at %s</a> · <a rel="up" href="../">lenses</a></nav>
<article class="lens" itemscope>
<p class="draft-label">A lens: someone's way of reading these plans. Not testimony.</p>
<h1 itemprop="title">%s</h1>
<meta itemprop="itemCount" content="%d">
<dl class="facts">
<dt>File</dt><dd><code itemprop="file">%s</code></dd>
<dt>Repository</dt>%s
</dl>
%s
<ul class="lens-items">
%s
</ul>
</article>`, Rel(here, ""), esc(s.RepoName), esc(s.Generated.Short), inline(l.Title), len(l.Items), esc(l.File), repo, intro, strings.Join(items, "\n")), "")
}

func short(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}
