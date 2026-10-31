package site

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

// The HTML is the API: what a person sees and what an agent fetches are the
// same pages. Structure is carried by rel links and microdata, whose names the
// profile page defines.

type Page struct {
	Href, Title, HTML string
}

type Guide struct {
	Markdown, Version string
}

func RenderSite(s *Site, guide Guide) []Page {
	pages := []Page{home(s), profile(s), guidePage(s, guide)}
	for _, p := range s.Plans {
		pages = append(pages, planPage(s, p))
		if p.AsBuilt != nil {
			pages = append(pages, asBuiltPage(s, p, p.AsBuilt))
		}
	}
	for _, pl := range s.Places {
		pages = append(pages, placePage(s, pl))
	}
	return pages
}

// --- pages ---

func home(s *Site) Page {
	here := ""
	title := fmt.Sprintf("%s at %s", s.RepoName, s.Generated.Short)
	var plans, places []string
	for _, p := range s.Plans {
		asBuilt, revisits := "", ""
		if p.AsBuilt != nil {
			asBuilt = fmt.Sprintf(`, with <a rel="as-built" itemprop="asBuilt" href="%s">an as-built</a>`, Rel(here, p.AsBuilt.Href))
		}
		if p.Revisits != nil {
			revisits = fmt.Sprintf(`, revisits <a rel="revisits" itemprop="revisits" href="%s">%s</a>`, Rel(here, p.Revisits.Href), esc(p.Revisits.Name))
		}
		plans = append(plans, fmt.Sprintf(`<li itemprop="plan" itemscope>
  <a rel="item" itemprop="url" href="%s"><span itemprop="title">%s</span></a>
  <span class="meta"><span itemprop="name">%s</span>, %s, base %s, %s%s%s</span>
</li>`, Rel(here, p.Href), inline(p.Doc.Doc.Title), esc(p.Name), whenHTML(p), baseHTML(p), entryCount(len(p.Doc.Entries)), asBuilt, revisits))
	}
	for _, pl := range s.Places {
		places = append(places, fmt.Sprintf(`<li itemprop="place" itemscope>
  <a rel="item" itemprop="url" href="%s"><code itemprop="name">%s</code></a>
  <span class="meta"><span itemprop="kind">%s</span>, %s</span>
</li>`, Rel(here, pl.Href), esc(pl.Label), pl.Kind, entryCount(len(pl.Refs))))
	}
	planList := fmt.Sprintf(`<div class="welcome"><p>No plans here yet. That's how every repository starts.</p>
<p>A plan is a markdown file in <code>plans/</code>, written before the work: what you decided, what you turned down, and what you're guessing, each with its reason. <a href="%s">The guide</a> says how, starting with “Write the plan.”</p>
<p>When you want to think one through first, put a draft at <code>drafts/plan.draft.md</code>. While <code>phoropter</code> is serving, it's shown at <code>draft/</code>.</p></div>`, Rel(here, "guide/"))
	if len(plans) > 0 {
		planList = `<ul class="list">` + strings.Join(plans, "\n") + `</ul>`
	}
	placeList := `<p>Nothing is anchored.</p>`
	if len(places) > 0 {
		placeList = `<ul class="list">` + strings.Join(places, "\n") + `</ul>`
	}
	return layout(s, here, title, fmt.Sprintf(`
<h1>%s</h1>
<div class="orientation">
<p>This is <strong>%s</strong> read through its plans: the plans and their as-built files in <code>plans/</code>, each anchored to the code it was about, at commit <code>%s</code>. Nothing here is written to; it’s generated from git.</p>
<p>To find out why some code is the way it is, start from a file or symbol under <em>Anchored places</em>, which lists the plans that touch it, newest first. To read the history, start from <em>Plans</em>, oldest first. The <a rel="help" href="%s">guide</a> says how plans are written, and the <a href="%s">profile</a> defines every link relation and microdata property on these pages.</p>
</div>
<div itemscope>
<section>
<h2>Plans</h2>
%s
</section>
<section>
<h2>Anchored places</h2>
<p>Each symbol or file a plan points at has a page listing the plans there.</p>
%s
</section>
</div>`, esc(title), esc(s.RepoName), esc(s.Generated.Short), Rel(here, "guide/"), Rel(here, "profile/"), planList, placeList), "")
}

func planPage(s *Site, p *Plan) Page {
	here := p.Href
	asBuilt := ""
	if p.AsBuilt != nil {
		asBuilt = fmt.Sprintf(`<dt>As built</dt><dd><a rel="as-built" itemprop="asBuilt" href="%s">%s</a></dd>`, Rel(here, p.AsBuilt.Href), esc(p.AsBuilt.File))
	}
	cart := ""
	if v, ok := p.Doc.Doc.FM("cart"); ok {
		cart = fmt.Sprintf(`<dt>Cart</dt><dd itemprop="cart">%s</dd>`, esc(v))
	}
	intro := ""
	if p.Doc.Doc.Intro != "" {
		intro = fmt.Sprintf(`<div class="intro" itemprop="intro">%s</div>`, block(p.Doc.Doc.Intro))
	}
	return layout(s, here, p.Doc.Doc.Title, fmt.Sprintf(`
<nav><a rel="collection" href="%s">%s at %s</a></nav>
<article itemscope>
<h1 itemprop="title">%s</h1>
<dl class="facts">
<dt>Name</dt><dd itemprop="name">%s</dd>
%s
<dt>Date</dt><dd>%s</dd>
<dt>Base</dt><dd>%s</dd>
<dt>File</dt><dd><code itemprop="file">%s</code></dd>
%s
%s
%s
</dl>
%s
%s
</article>`, Rel(here, ""), esc(s.RepoName), esc(s.Generated.Short), inline(p.Doc.Doc.Title), esc(p.Name), cart, whenHTML(p), baseHTML(p), esc(p.File), asBuilt, revisitsHTML(here, p), frontmatterRest(p.Doc.Doc, "cart", "base", "revisits"), intro, entries(here, &p.Doc, "")), "")
}

func asBuiltPage(s *Site, p *Plan, d *Doc) Page {
	here := d.Href
	head := ""
	if d.Before != nil {
		head = fmt.Sprintf(`<dt>Head</dt><dd><data itemprop="head" value="%s"><code>%s</code></data></dd>`, d.Before.SHA, esc(d.Before.Short))
	} else if d.BeforeText != "" {
		head = fmt.Sprintf(`<dt>Head</dt><dd itemprop="head">%s</dd>`, esc(d.BeforeText))
	}
	intro := ""
	if d.Doc.Intro != "" {
		intro = fmt.Sprintf(`<div class="intro" itemprop="intro">%s</div>`, block(d.Doc.Intro))
	}
	return layout(s, here, d.Doc.Title, fmt.Sprintf(`
<nav><a rel="up" href="%s">%s</a></nav>
<article itemscope>
<h1 itemprop="title">%s</h1>
<dl class="facts">
<dt>Plan</dt><dd><a rel="up" itemprop="plan" href="%s">%s</a></dd>
%s
<dt>File</dt><dd><code itemprop="file">%s</code></dd>
%s
</dl>
%s
%s
</article>`, Rel(here, p.Href), inline(p.Doc.Doc.Title), inline(d.Doc.Title), Rel(here, p.Href), esc(p.File), head, esc(d.File), frontmatterRest(d.Doc, "head"), intro, entries(here, d, "")), "")
}

func entries(here string, d *Doc, _ string) string {
	var out []string
	for _, e := range d.Entries {
		out = append(out, entryHTML(here, d, e))
	}
	return strings.Join(out, "\n")
}

func placePage(s *Site, pl *Place) Page {
	here := pl.Href
	location := ""
	if def := pl.Definition; def != nil {
		location = fmt.Sprintf(`<dt>At %s</dt><dd><code itemprop="file">%s</code>, lines <span itemprop="startLine">%d</span>–<span itemprop="endLine">%d</span></dd>`, esc(s.Generated.Short), esc(def.File), def.StartLine, def.EndLine)
	} else if pl.Kind == "symbol" {
		location = fmt.Sprintf(`<dt>At %s</dt><dd>not in the index</dd>`, esc(s.Generated.Short))
	}
	var refs []string
	for _, r := range pl.Refs {
		kind := ""
		if r.Entry.Kind != "" {
			kind = fmt.Sprintf(`<span itemprop="kind">%s</span>: `, esc(r.Entry.Kind))
		}
		ab := ""
		if r.Doc != &r.Plan.Doc {
			ab = " (as-built)"
		}
		isNew := ""
		if r.Anchor.IsNew {
			isNew = `, new<meta itemprop="new" content="true">`
		}
		refs = append(refs, fmt.Sprintf(`<li itemprop="anchoredBy" itemscope>
  <a rel="anchored-by" itemprop="url" href="%s#%s">%s<span itemprop="title">%s</span></a>
  <span class="meta">in <span itemprop="plan">%s</span>, %s%s%s</span>
  <code class="where" itemprop="where">%s</code>
</li>`, Rel(here, r.Doc.Href), r.Entry.Slug, kind, inline(r.Entry.Title), esc(r.Plan.Name), whenHTML(r.Plan), ab, isNew, esc(r.Anchor.Where)))
	}
	code := ""
	if pl.Code != nil {
		code = fmt.Sprintf(`<div class="view">%s</div>`, hunkHTML(*pl.Code))
	}
	label := "Path"
	if pl.Kind == "symbol" {
		label = "Symbol"
	}
	return layout(s, here, pl.Label, fmt.Sprintf(`
<nav><a rel="collection" href="%s">%s at %s</a></nav>
<article itemscope>
<h1><code itemprop="name">%s</code></h1>
<dl class="facts">
<dt>Kind</dt><dd itemprop="kind">%s</dd>
<dt>%s</dt><dd><code itemprop="%s">%s</code></dd>
%s
</dl>
%s
<h2>Plans anchored here</h2>
<ul class="list">%s</ul>
</article>`, Rel(here, ""), esc(s.RepoName), esc(s.Generated.Short), esc(pl.Label), pl.Kind, label, pl.Kind, esc(pl.Full), location, code, strings.Join(refs, "\n")), "")
}

var h1RE = regexp.MustCompile(`(?s)<h1[^>]*>(.*?)</h1>`)

// guidePage is the same guide on every site. Its own links are relative to guide/.
func guidePage(s *Site, g Guide) Page {
	here := "guide/"
	h := block(g.Markdown)
	title := "Guide"
	if m := h1RE.FindStringSubmatch(h); m != nil {
		title = m[1]
	}
	version := fmt.Sprintf(`<p class="meta">From phoropter, %s. It describes the tool, so it’s the same on every site phoropter generates.</p>`, esc(g.Version))
	h = strings.Replace(h, "</h1>", "</h1>\n"+version, 1)
	return layout(s, here, title, fmt.Sprintf(`
<nav><a href="%s">%s at %s</a></nav>
<article class="guide">
%s
</article>`, Rel(here, ""), esc(s.RepoName), esc(s.Generated.Short), h), "")
}

// DraftPage is the one draft, rendered per request by the dev server (loden).
// It says it's the working tree, not a commit; its anchors still resolve at
// commits.
func DraftPage(s *Site, d *Draft) Page {
	here := "draft/"
	p := d.Plan
	at := d.ReadAt.UTC().Format("2006-01-02T15:04:05.000Z")
	header := fmt.Sprintf(`Draft, not committed: <code itemprop="file">%s</code> in <span itemprop="repository">%s</span>,
read at <time itemprop="readAt" datetime="%s">%s UTC</time>
<span class="meta">sha256 <data itemprop="sha256" value="%s">%s</data>; anchors at <data itemprop="commit" value="%s">%s</data></span>`, esc(p.File), esc(s.RepoName), at, at[11:19], d.SHA256, d.SHA256[:12], s.Generated.SHA, esc(s.Generated.Short))
	var anns []Annotation
	if d.Notes != nil {
		anns = d.Notes.Annotations
	}
	extras := func(key string) string {
		var b strings.Builder
		for _, pr := range d.Prompts[key] {
			if b.Len() > 0 {
				b.WriteString("\n")
			}
			b.WriteString(promptHTML(pr))
		}
		var notes []string
		for _, a := range anns {
			if a.Found && a.At == key {
				notes = append(notes, AnnotationHTML(a))
			}
		}
		return b.String() + strings.Join(notes, "\n")
	}
	var top []string
	for _, a := range anns {
		if !a.HasQuote || !a.Found {
			top = append(top, AnnotationHTML(a))
		}
	}
	notesFact := `<dt>Notes</dt><dd class="note">none yet</dd>`
	if d.Notes != nil {
		notesFact = fmt.Sprintf(`<dt>Notes</dt><dd><code>plan.draft.notes.md</code>, sha256 <data itemprop="notesSha256" value="%s">%s</data>%s</dd>`, d.Notes.SHA256, d.Notes.SHA256[:12], notesAnnotate(d))
	}
	title := p.Doc.Doc.Title
	if title == "" {
		title = "Draft"
	}
	h1 := p.Doc.Doc.Title
	if h1 == "" {
		h1 = "Untitled draft"
	}
	intro := ""
	if p.Doc.Doc.Intro != "" {
		intro = fmt.Sprintf(`<div class="intro" itemprop="intro">%s</div>`, block(p.Doc.Doc.Intro))
	}
	var es []string
	for _, e := range p.Doc.Entries {
		es = append(es, entryHTML(here, &p.Doc, e)+extras(e.Slug))
	}
	return layout(s, here, title, fmt.Sprintf(`
<nav><a href="%s">%s at %s</a></nav>
<article class="draft" itemscope>
<p class="draft-label">Draft. Nothing here is testimony yet.</p>
<h1 itemprop="title">%s</h1>
<dl class="facts">
<dt>Base</dt><dd>%s</dd>
%s
%s
</dl>
%s
%s
%s
%s
</article>`, Rel(here, ""), esc(s.RepoName), esc(s.Generated.Short), inline(h1), baseHTML(p), revisitsHTML(here, p), notesFact, strings.Join(top, "\n"), intro, extras("intro"), strings.Join(es, "\n")), header)
}

// notesAnnotate: notes say which version of the draft they annotate. When it
// isn't this one, their quotes may have come loose.
func notesAnnotate(d *Draft) string {
	if d.Notes == nil || d.Notes.Annotates == "" {
		return ""
	}
	if d.Notes.Current {
		return ", annotating this version"
	}
	a := d.Notes.Annotates
	return fmt.Sprintf(`, annotating <data itemprop="annotates" value="%s">%s</data> <span class="note">an earlier version of the draft; quotes may have come loose</span>`, a, a)
}

func promptHTML(p *Prompt) string {
	options := ""
	if len(p.Options) > 0 {
		var lis []string
		for _, o := range p.Options {
			lis = append(lis, fmt.Sprintf(`<li itemprop="option">%s</li>`, inline(o)))
		}
		options = "<ul>" + strings.Join(lis, "") + "</ul>"
	}
	fallback := `<p class="fallback note">No fallback given, so silence doesn’t say what happens.</p>`
	if p.Fallback != "" {
		fallback = fmt.Sprintf(`<p class="fallback"><strong>If unanswered:</strong> <span itemprop="fallback">%s</span></p>`, inline(p.Fallback))
	}
	answer := `<p class="note">Not answered yet.</p>`
	if p.Answer != nil {
		answer = fmt.Sprintf(`<div class="answer" itemprop="answer">%s</div>`, block(p.Answer.Comment))
	}
	body := ""
	if p.Body != "" {
		body = block(p.Body)
	}
	return fmt.Sprintf(`<div class="prompt" itemprop="prompt" itemscope>
<p class="prompt-heading"><span class="kind" itemprop="kind">%s</span>: <span itemprop="text">%s</span></p>
%s%s
%s
%s
</div>`, esc(p.Kind), inline(p.Text), body, options, fallback, answer)
}

func AnnotationHTML(a Annotation) string {
	missing, quote, comment := "", "", ""
	if a.HasQuote && !a.Found {
		missing = `<p class="note">quote not found: it moved under the note</p>`
	}
	if a.HasQuote {
		quote = fmt.Sprintf(`<blockquote itemprop="quote">%s</blockquote>`, esc(a.Quote))
	}
	if a.Comment != "" {
		comment = fmt.Sprintf(`<div itemprop="comment">%s</div>`, block(a.Comment))
	}
	return fmt.Sprintf(`<aside class="annotation" itemprop="annotation" itemscope>
<p class="meta" itemprop="heading">%s</p>
%s%s
%s
</aside>`, esc(a.Heading), quote, missing, comment)
}

// --- parts ---

func entryHTML(here string, d *Doc, e *ResolvedEntry) string {
	heading := fmt.Sprintf(`<span itemprop="title">%s</span>`, inline(e.Title))
	if e.Kind != "" {
		heading = fmt.Sprintf(`<span class="kind kind-%s" itemprop="kind">%s</span>: <span itemprop="title">%s</span>`, strings.ToLower(e.Kind), esc(e.Kind), inline(e.Title))
	}
	var anchors []string
	for _, a := range e.Resolved {
		anchors = append(anchors, anchorHTML(here, d, a))
	}
	reason := ""
	if e.Body != "" {
		reason = fmt.Sprintf(`<div class="reason" itemprop="reason">%s</div>`, withSource(block(e.Body)))
	}
	return fmt.Sprintf(`<section class="entry" id="%s" itemprop="entry" itemscope>
<h2><a class="self" href="#%s">%s</a></h2>
%s
%s
%s
%s
</section>`, e.Slug, e.Slug, heading, pairHTML(here, e), anchoredByHTML(here, e), strings.Join(anchors, "\n"), reason)
}

var sourceLeadRE = regexp.MustCompile(`^<p><strong>(Because|Answer) \((.*?)\):</strong>`)

// withSource: the source stays as written, inside the bold lead where the
// reader sees it; the span only makes it findable.
func withSource(h string) string {
	return sourceLeadRE.ReplaceAllString(h, `<p><strong>$1 (<span itemprop="source">$2</span>):</strong>`)
}

func pairHTML(here string, e *ResolvedEntry) string {
	link := func(r, prop string, to EntryRef, text string) string {
		return fmt.Sprintf(`<a rel="%s" itemprop="%s" href="%s#%s">%s</a>`, r, prop, Rel(here, to.Doc.Href), to.Entry.Slug, text)
	}
	if e.Kind == "Answered" {
		v := verdictHTML(e)
		if e.Answers != nil {
			return fmt.Sprintf(`<p class="pair">Answers %s: %s.</p>`, link("answers", "answers", *e.Answers, "the plan’s guess"), v)
		}
		return fmt.Sprintf(`<p class="pair">%s. <span class="note">No guess in the plan has this title.</span></p>`, v)
	}
	if len(e.AnsweredBy) > 0 {
		var parts []string
		for _, a := range e.AnsweredBy {
			parts = append(parts, fmt.Sprintf("%s: %s", link("answered-by", "answeredBy", a, "the as-built"), verdictHTML(a.Entry)))
		}
		return fmt.Sprintf(`<p class="pair">Answered in %s.</p>`, strings.Join(parts, ", "))
	}
	return ""
}

// verdictHTML is grommet's opening word, or a plain statement that there
// isn't one. The qualifiers after it stay in the prose.
func verdictHTML(a *ResolvedEntry) string {
	if a.Verdict == "" {
		return `<span class="note">no verdict (it doesn’t open with held, disproved, partly disproved, or not tested)</span>`
	}
	return fmt.Sprintf(`<strong class="verdict verdict-%s" itemprop="verdict">%s</strong>`, strings.Replace(a.Verdict, " ", "-", 1), a.Verdict)
}

func anchoredByHTML(here string, e *ResolvedEntry) string {
	if len(e.AnchoredBy) == 0 {
		return ""
	}
	var items []string
	for _, r := range e.AnchoredBy {
		items = append(items, fmt.Sprintf(`<a rel="anchored-by" itemprop="anchoredBy" href="%s#%s">%s, %s</a>`, Rel(here, r.Doc.Href), r.Entry.Slug, docName(r), kindTitle(r.Entry)))
	}
	return fmt.Sprintf(`<p class="pair">Anchored by %s.</p>`, strings.Join(items, "; "))
}

func revisitsHTML(here string, p *Plan) string {
	text, has := p.Doc.Doc.FM("revisits")
	var out []string
	if p.Revisits != nil {
		out = append(out, fmt.Sprintf(`<dt>Revisits</dt><dd><a rel="revisits" itemprop="revisits" href="%s">%s</a> <span class="meta">%s</span></dd>`, Rel(here, p.Revisits.Href), inline(p.Revisits.Doc.Doc.Title), esc(p.Revisits.Name)))
	} else if has {
		out = append(out, fmt.Sprintf(`<dt>Revisits</dt><dd><code itemprop="revisits">%s</code> <span class="note">not a plan on this site</span></dd>`, esc(text)))
	}
	if len(p.RevisitedBy) > 0 {
		var links []string
		for _, q := range p.RevisitedBy {
			links = append(links, fmt.Sprintf(`<a rel="revisited-by" itemprop="revisitedBy" href="%s">%s</a> <span class="meta">%s</span>`, Rel(here, q.Href), inline(q.Doc.Doc.Title), esc(q.Name)))
		}
		out = append(out, fmt.Sprintf(`<dt>Revisited by</dt><dd>%s</dd>`, strings.Join(links, "<br>")))
	}
	return strings.Join(out, "\n")
}

func docName(r EntryRef) string {
	if r.Doc == &r.Plan.Doc {
		return esc(r.Plan.Name)
	}
	return esc(r.Plan.Name + "'s as-built")
}

func kindTitle(e *ResolvedEntry) string {
	if e.Kind != "" {
		return fmt.Sprintf("%s: %s", esc(e.Kind), inline(e.Title))
	}
	return inline(e.Title)
}

func anchorHTML(here string, d *Doc, ra *ResolvedAnchor) string {
	a := ra.Anchor
	label := "where"
	if a.IsNew {
		label = "where (new)"
	}
	place, target, quote, note, views, isNew := "", "", "", "", "", ""
	if ra.Place != nil {
		place = fmt.Sprintf(` <a rel="anchored-at" itemprop="place" href="%s">%s page</a>`, Rel(here, ra.Place.Href), ra.Place.Kind)
	}
	if ra.Entry != nil {
		target = fmt.Sprintf(`<p class="target"><span class="label">entry</span> <a rel="anchored-at" itemprop="target" href="%s#%s">%s</a> <span class="meta">in %s</span></p>`, Rel(here, ra.Entry.Doc.Href), ra.Entry.Entry.Slug, kindTitle(ra.Entry.Entry), docName(*ra.Entry))
	} else if ra.Plan != nil {
		target = fmt.Sprintf(`<p class="target"><span class="label">plan</span> <a rel="anchored-at" itemprop="target" href="%s">%s</a> <span class="meta">%s</span></p>`, Rel(here, ra.Plan.Href), inline(ra.Plan.Doc.Doc.Title), esc(ra.Plan.Name))
	}
	if a.HasQuote {
		quote = fmt.Sprintf(`<p class="quote"><span class="label">quote</span> <code itemprop="quote">%s</code></p>`, esc(a.Quote))
	}
	if ra.Note != "" {
		note = fmt.Sprintf(`<p class="note" itemprop="note">%s</p>`, esc(ra.Note))
	}
	if len(ra.Views) > 0 {
		var vs []string
		for _, v := range ra.Views {
			vs = append(vs, viewHTML(v))
		}
		views = `<div class="views">` + strings.Join(vs, "\n") + `</div>`
	}
	if a.IsNew {
		isNew = `<meta itemprop="new" content="true">`
	}
	return fmt.Sprintf(`<div class="anchor" itemprop="anchor" itemscope>
%s
<p class="where"><span class="label">%s</span> <code itemprop="where">%s</code>%s</p>
%s%s%s%s
</div>`, isNew, label, esc(a.Where), place, target, quote, note, views)
}

func viewHTML(v *View) string {
	rng, file, sameAs := "", "", ""
	if v.StartLine != 0 {
		rng = fmt.Sprintf(`<meta itemprop="startLine" content="%d"><meta itemprop="endLine" content="%d">`, v.StartLine, v.EndLine)
	}
	if v.File != "" {
		file = fmt.Sprintf(`<code class="file" itemprop="file">%s</code>`, esc(v.File))
	}
	if v.SameAs != "" {
		sameAs = fmt.Sprintf(`<meta itemprop="sameAs" content="%s">`, v.SameAs)
	}
	var matches, code []string
	for _, m := range v.At {
		matches = append(matches, matchHTML(m))
	}
	for _, h := range v.Hunks {
		code = append(code, hunkHTML(h))
	}
	return fmt.Sprintf(`<figure class="view status-%s" itemprop="view" itemscope>
<figcaption>
<span class="role" itemprop="role">%s</span>
<data itemprop="commit" value="%s"><code>%s</code></data>
<span class="status" itemprop="status">%s</span>
%s
<meta itemprop="matches" content="%d">%s%s%s
<span class="note" itemprop="note">%s</span>
</figcaption>
%s
</figure>`, v.Status, esc(v.Role), v.SHA, esc(v.Short), v.Status, file, v.Matches, strings.Join(matches, ""), rng, sameAs, inline(v.Note), strings.Join(code, "\n"))
}

// matchHTML: one item per match, so a reader finds the quote's lines without
// the note.
func matchHTML(lines []int) string {
	return fmt.Sprintf(`<span itemprop="match" itemscope hidden><meta itemprop="startLine" content="%d"><meta itemprop="endLine" content="%d"></span>`, lines[0], lines[len(lines)-1])
}

// hunkHTML: codeStart comes just before the code it numbers; microdata keeps
// document order, so the nth codeStart belongs to the nth code. The numbers
// are a CSS counter from it, so code's text is just the code.
func hunkHTML(h Hunk) string {
	marked := map[int]bool{}
	for _, n := range h.Marked {
		marked[n] = true
	}
	lines := make([]string, len(h.Lines))
	for i, line := range h.Lines {
		if marked[h.Start+i] {
			lines[i] = fmt.Sprintf(`<mark class="line">%s</mark>`, esc(line))
		} else {
			lines[i] = fmt.Sprintf(`<span class="line">%s</span>`, esc(line))
		}
	}
	return fmt.Sprintf(`<meta itemprop="codeStart" content="%d"><pre itemprop="code" data-start="%d"><code style="counter-reset: line %d">%s</code></pre>`, h.Start, h.Start, h.Start-1, strings.Join(lines, "\n"))
}

func layout(s *Site, here, title, main, header string) Page {
	// The home page's title already is the provenance, so it isn't said twice.
	provenance := fmt.Sprintf("%s at %s", s.RepoName, s.Generated.Short)
	if header == "" {
		header = fmt.Sprintf(`Generated from <span itemprop="repository">%s</span>
at <data itemprop="commit" value="%s"><code>%s</code></data>
<span class="meta">%s</span>`, esc(s.RepoName), s.Generated.SHA, esc(s.Generated.Short), s.Generated.SHA)
	}
	pageTitle := provenance
	if here != "" {
		pageTitle = strings.ReplaceAll(title, "`", "") + " · " + provenance
	}
	return Page{Href: here, Title: title, HTML: fmt.Sprintf(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>%s</title>
<link rel="profile" href="%s">
<link rel="help" href="%s">
<link rel="stylesheet" href="%s">
</head>
<body>
<header class="provenance" itemscope>
%s
<span class="links"><a class="home-link" href="%s">home</a>
<a rel="help" href="%s">guide</a>
<a class="profile-link" href="%s">profile</a></span>
</header>
<main>
%s
</main>
</body>
</html>
`, esc(pageTitle), Rel(here, "profile/"), Rel(here, "guide/"), Rel(here, "style.css"), header, Rel(here, ""), Rel(here, "guide/"), Rel(here, "profile/"), main)}
}

// whenHTML is the filename's date, and the commit that added the plan, which
// is what orders plans that share a date.
func whenHTML(p *Plan) string {
	date := "no date"
	if p.Date != "" {
		date = fmt.Sprintf(`<time itemprop="date" datetime="%s">%s</time>`, p.Date, p.Date)
	}
	if p.Added == nil {
		return date
	}
	at := time.Unix(p.Added.Time, 0).UTC().Format("2006-01-02T15:04:05.000Z")
	return fmt.Sprintf(`%s, added in <data itemprop="added" value="%s"><code>%s</code></data> <time datetime="%s">%s UTC</time>`, date, p.Added.SHA, esc(p.Added.Short), at, at[11:16])
}

func baseHTML(p *Plan) string {
	if p.Before != nil {
		return fmt.Sprintf(`<data itemprop="base" value="%s"><code>%s</code></data>`, p.Before.SHA, esc(p.Before.Short))
	}
	text := p.BeforeText
	if text == "" {
		text = "not given"
	}
	return fmt.Sprintf(`<span itemprop="base">%s</span>`, esc(text))
}

func frontmatterRest(d *PlanDoc, shown ...string) string {
	var out []string
outer:
	for _, kv := range d.Frontmatter {
		for _, s := range shown {
			if kv.Key == s {
				continue outer
			}
		}
		out = append(out, fmt.Sprintf(`<dt>%s</dt><dd>%s</dd>`, esc(strings.ToUpper(kv.Key[:1])+kv.Key[1:]), esc(kv.Value)))
	}
	return strings.Join(out, "\n")
}

// Rel is a relative href from one site directory to another, so the site
// works from any root, including file://.
func Rel(from, to string) string {
	depth := 0
	for _, part := range strings.Split(from, "/") {
		if part != "" {
			depth++
		}
	}
	p := strings.Repeat("../", depth) + encodeURI(to)
	if p == "" {
		return "./"
	}
	return p
}

// encodeURI is JavaScript's: it escapes everything but letters, digits and
// ;,/?:@&=+$-_.!~*'()#.
func encodeURI(s string) string {
	const keep = ";,/?:@&=+$-_.!~*'()#"
	var b strings.Builder
	for _, c := range []byte(s) {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || strings.IndexByte(keep, c) >= 0 {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}

var md = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithRendererOptions(html.WithUnsafe()),
)

// inlineMD parses with only the paragraph block parser, so a title like
// "1. Install" stays text instead of becoming a list, as marked's parseInline
// leaves it.
var inlineMD = goldmark.New(
	goldmark.WithExtensions(extension.Strikethrough, extension.Linkify),
	goldmark.WithParserOptions(parser.WithBlockParsers(util.Prioritized(parser.NewParagraphParser(), 1000))),
	goldmark.WithRendererOptions(html.WithUnsafe()),
)

func block(src string) string {
	var b bytes.Buffer
	md.Convert([]byte(src), &b)
	return b.String()
}

func inline(src string) string {
	var b bytes.Buffer
	inlineMD.Convert([]byte(src), &b)
	out := strings.TrimSuffix(b.String(), "\n")
	out = strings.TrimPrefix(out, "<p>")
	return strings.TrimSuffix(out, "</p>")
}

func entryCount(n int) string {
	word := "entries"
	if n == 1 {
		word = "entry"
	}
	return fmt.Sprintf(`<data itemprop="entryCount" value="%d">%d %s</data>`, n, n, word)
}

func esc(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;").Replace(s)
}
