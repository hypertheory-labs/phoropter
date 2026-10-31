// Package substrate turns phoropter's pages into typed values, reading only
// what the profile defines where it can, and noting in a Ledger each place it
// can't.
package substrate

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/net/html"

	"phoropter-tui/internal/microdata"
)

// Ref is a commit as the page gave it. SHA is empty when the page gave text
// instead, like a base of "none, a new repo".
type Ref struct {
	SHA   string
	Label string
}

func (r Ref) String() string { return r.Label }

type Link struct {
	URL   string
	Label string
}

type Home struct {
	URL        string
	Repository string
	Commit     Ref
	DraftURL   string // the repository's one draft, when the dev server links it
	Plans      []PlanSummary
	Places     []PlaceSummary
}

type PlanSummary struct {
	URL, Title, Name, Date string
	Added, Base            Ref
	Entries                int
	AsBuiltURL             string
	RevisitsURL            string
}

type PlaceSummary struct {
	URL, Name, Kind string
	Entries         int
}

// Doc is a plan page, an as-built page, or the draft page; they share a shape.
type Doc struct {
	URL         string
	IsAsBuilt   bool
	Draft       *Draft // nil unless this is the draft
	AnchorsAt   Ref    // a draft's: the commit its anchors resolve at
	Title       string
	Name        string
	Cart        string
	Date        string
	File        string
	Added       Ref
	Base        Ref // a plan's
	Head        Ref // an as-built's
	AsBuiltURL  string
	PlanURL     string // an as-built's plan
	Revisits    *Link
	RevisitedBy []Link
	Intro       string
	Facts       [][2]string // the facts list as shown, for display
	Entries     []Entry
	Prompts     []Prompt     // a draft's open questions
	Annotations []Annotation // a draft's notes
}

// Draft is what a draft page's header says: it's the working tree, read at a
// time, not a commit.
type Draft struct {
	File, ReadAt, SHA256 string
}

// A Prompt is a draft's open question. After is the ID of the entry it
// follows, or "" when it comes before them all; so is an Annotation's.
type Prompt struct {
	After, Kind, Text string
	Options           []string
	Fallback, Answer  string
}

type Annotation struct {
	After, Heading, Quote, Comment string
}

type Entry struct {
	ID, Kind, Title string
	Source, Verdict string
	Reason          string
	Anchors         []Anchor
	Answers         *Link
	AnsweredBy      []Link
	AnchoredBy      []Link
}

type Anchor struct {
	Where, Quote, Note string
	New                bool
	Place              *Link // a symbol or file page
	Target             *Link // another plan's entry, or a whole plan
	Views              []View
}

type View struct {
	Role, Status, File, Note string
	Commit                   Ref
	Matches                  int
	StartLine, EndLine       int
	Match                    []Range // the lines each match covers
	MatchLine                int     // the first match's first line, or 0
	SameAs                   string
	Code                     *Excerpt
	CodeFrom                 string // the commit the code was copied from, for sameAs
}

type Range struct{ Start, End int }

type Excerpt struct {
	Start int
	Lines []Line
}

type Line struct {
	N      int
	Text   string
	Marked bool
}

type Place struct {
	URL, Name, Kind, File, Symbol string
	StartLine, EndLine            int
	Code                          *Excerpt
	FileURL                       string // for a symbol, where its file's page would be
	AnchoredBy                    []PlaceRef
}

type PlaceRef struct {
	URL, Kind, Title, Plan, Date, Where string
	Added                               Ref
	New                                 bool
}

// Page is any page, with its provenance and the top-level item in <main>.
// Commit is the commit the page was generated from. A draft page wasn't
// generated from one: it has Draft, and its header's commit is AnchorsAt.
type Page struct {
	URL        string
	Doc        *microdata.Document
	Repository string
	Commit     Ref
	Draft      *Draft
	AnchorsAt  Ref
	DraftURL   string // rel="draft"
	Main       *microdata.Item
}

func (c *Client) Page(ref string) (*Page, error) {
	u := c.Resolve(ref)
	d, err := c.Fetch(u)
	if err != nil {
		return nil, err
	}
	p := &Page{URL: stripFragment(u), Doc: d}
	for _, it := range d.Items {
		switch {
		case it.Node.Data == "header" || microdata.Within(it.Node, "header"):
			p.Repository = it.Str("repository")
			if it.Get("readAt") != nil {
				p.Draft = &Draft{File: it.Str("file"), ReadAt: it.Str("readAt"), SHA256: it.Str("sha256")}
				p.AnchorsAt = refOf(it.Get("commit"))
			} else {
				p.Commit = refOf(it.Get("commit"))
			}
		case p.Main == nil && microdata.Within(it.Node, "main"):
			p.Main = it
		}
	}
	for _, l := range d.Links {
		if slices.Contains(l.Rel, "draft") {
			p.DraftURL = l.Href
		}
	}
	if p.Main == nil {
		return nil, fmt.Errorf("%s: no item in <main>", u)
	}
	return p, nil
}

func (c *Client) Home() (*Home, error) {
	p, err := c.Page("")
	if err != nil {
		return nil, err
	}
	h := &Home{URL: p.URL, Repository: p.Repository, Commit: p.Commit, DraftURL: p.DraftURL}
	for _, it := range p.Main.Items("plan") {
		s := PlanSummary{
			URL:         it.Str("url"),
			Title:       prose(it.Get("title")),
			Name:        it.Str("name"),
			Date:        it.Str("date"),
			Added:       refOf(it.Get("added")),
			Base:        refOf(it.Get("base")),
			AsBuiltURL:  it.Str("asBuilt"),
			RevisitsURL: it.Str("revisits"),
		}
		s.Entries, _ = strconv.Atoi(it.Str("entryCount"))
		h.Plans = append(h.Plans, s)
	}
	for _, it := range p.Main.Items("place") {
		s := PlaceSummary{URL: it.Str("url"), Name: it.Str("name"), Kind: it.Str("kind")}
		s.Entries, _ = strconv.Atoi(it.Str("entryCount"))
		h.Places = append(h.Places, s)
	}
	return h, nil
}

// Doc reads a plan or as-built page.
func (c *Client) Doc(ref string) (*Doc, error) {
	p, err := c.Page(ref)
	if err != nil {
		return nil, err
	}
	it := p.Main
	d := &Doc{
		URL:   p.URL,
		Title: prose(it.Get("title")),
		Name:  it.Str("name"),
		Cart:  it.Str("cart"),
		Date:  it.Str("date"),
		File:  it.Str("file"),
		Added: refOf(it.Get("added")),
		Intro: prose(it.Get("intro")),
		Facts: facts(it.Node),
	}
	if v := it.Get("head"); v != nil {
		d.IsAsBuilt = true
		d.Head = refOf(v)
	}
	if v := it.Get("plan"); v != nil {
		d.IsAsBuilt = true
		d.PlanURL = v.Text
	}
	if v := it.Get("base"); v != nil {
		d.Base = refOf(v)
	}
	if v := it.Get("asBuilt"); v != nil {
		d.AsBuiltURL = v.Text
	}
	if v := it.Get("revisits"); v != nil {
		d.Revisits = &Link{URL: v.Text, Label: prose(v)}
	}
	for _, v := range it.Props["revisitedBy"] {
		d.RevisitedBy = append(d.RevisitedBy, Link{URL: v.Text, Label: prose(v)})
	}
	for _, e := range it.Items("entry") {
		d.Entries = append(d.Entries, c.entry(e, p.URL))
	}
	if p.Draft != nil {
		d.Draft, d.AnchorsAt, d.Name = p.Draft, p.AnchorsAt, "draft"
		c.draftExtras(d, it, p)
	}
	return d, nil
}

// draftExtras reads a draft's prompts and annotations. Each belongs to the
// entry it follows in the page, so it's placed by tree order.
func (c *Client) draftExtras(d *Doc, it *microdata.Item, p *Page) {
	root := p.Doc.Root
	pos := map[*html.Node]int{}
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		pos[n] = len(pos)
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			walk(ch)
		}
	}
	walk(root)
	after := func(n *html.Node) string {
		id := ""
		for _, e := range it.Items("entry") {
			if pos[e.Node] < pos[n] {
				id = e.ID
			}
		}
		return id
	}
	for _, v := range it.Items("prompt") {
		pr := Prompt{
			After:    after(v.Node),
			Kind:     v.Str("kind"),
			Text:     prose(v.Get("text")),
			Fallback: prose(v.Get("fallback")),
			Answer:   prose(v.Get("answer")),
		}
		if a := v.Get("answer"); a != nil && hasCheckbox(a.Node) {
			c.Ledger.Note("answer-choices-unprofiled", p.URL, "%s: %s", pr.Kind, pr.Text)
		}
		for _, o := range v.Props["option"] {
			pr.Options = append(pr.Options, prose(o))
		}
		d.Prompts = append(d.Prompts, pr)
	}
	for _, v := range it.Items("annotation") {
		d.Annotations = append(d.Annotations, Annotation{
			After:   after(v.Node),
			Heading: v.Str("heading"),
			Quote:   v.Str("quote"),
			Comment: prose(v.Get("comment")),
		})
	}
}

func (c *Client) entry(it *microdata.Item, page string) Entry {
	e := Entry{
		ID:      it.ID,
		Kind:    it.Str("kind"),
		Title:   prose(it.Get("title")),
		Source:  it.Str("source"),
		Verdict: it.Str("verdict"),
		Reason:  prose(it.Get("reason")),
	}
	if v := it.Get("answers"); v != nil {
		e.Answers = &Link{URL: v.Text, Label: prose(v)}
	}
	for _, v := range it.Props["answeredBy"] {
		e.AnsweredBy = append(e.AnsweredBy, Link{URL: v.Text, Label: linkSentence(v)})
	}
	for _, v := range it.Props["anchoredBy"] {
		e.AnchoredBy = append(e.AnchoredBy, Link{URL: v.Text, Label: prose(v)})
	}
	for _, a := range it.Items("anchor") {
		e.Anchors = append(e.Anchors, c.anchor(a, page))
	}
	return e
}

func (c *Client) anchor(it *microdata.Item, page string) Anchor {
	a := Anchor{
		Where: it.Str("where"),
		Quote: it.Str("quote"),
		Note:  it.Str("note"),
		New:   it.Str("new") == "true",
	}
	if v := it.Get("place"); v != nil {
		a.Place = &Link{URL: v.Text, Label: prose(v)}
	}
	if v := it.Get("target"); v != nil {
		a.Target = &Link{URL: v.Text, Label: prose(v)}
	}
	for _, v := range it.Items("view") {
		a.Views = append(a.Views, c.view(v, page))
	}
	// sameAs names the commit whose code this view repeats.
	for i := range a.Views {
		v := &a.Views[i]
		if v.SameAs == "" || v.Code != nil {
			continue
		}
		for _, w := range a.Views {
			if w.Code != nil && strings.HasPrefix(w.Commit.SHA, v.SameAs) {
				v.Code, v.CodeFrom = w.Code, w.Commit.Label
				if v.MatchLine == 0 {
					v.MatchLine = w.MatchLine
				}
				c.Ledger.Note("same-as-deref", page, "%s view, same as %s", v.Role, v.SameAs)
			}
		}
	}
	for _, v := range a.Views {
		if v.Code == nil && v.Status != "no-quote" {
			c.Ledger.Note("view-no-code", page, "%s view of %s: %s", v.Role, a.Where, v.Status)
		}
	}
	if a.New && len(a.Views) == 0 {
		c.Ledger.Note("new-never-resolved", page, "%s", a.Where)
	}
	if !a.New && a.Place != nil && strings.Contains(a.Where, "›") && strings.Contains(a.Place.URL, "/files/") {
		c.Ledger.Note("where-path-unresolved", page, "%s", a.Where)
	}
	return a
}

func (c *Client) view(it *microdata.Item, page string) View {
	v := View{
		Role:   it.Str("role"),
		Status: it.Str("status"),
		File:   it.Str("file"),
		Note:   it.Str("note"),
		Commit: refOf(it.Get("commit")),
		SameAs: it.Str("sameAs"),
	}
	v.Matches, _ = strconv.Atoi(it.Str("matches"))
	v.StartLine, _ = strconv.Atoi(it.Str("startLine"))
	v.EndLine, _ = strconv.Atoi(it.Str("endLine"))
	for _, m := range it.Items("match") {
		var r Range
		r.Start, _ = strconv.Atoi(m.Str("startLine"))
		r.End, _ = strconv.Atoi(m.Str("endLine"))
		v.Match = append(v.Match, r)
	}
	if len(v.Match) > 0 {
		v.MatchLine = v.Match[0].Start
	}
	if codes := c.codes(it); len(codes) > 0 {
		v.Code = codes[0]
		for i := range v.Code.Lines {
			l := &v.Code.Lines[i]
			for _, r := range v.Match {
				l.Marked = l.Marked || (r.Start <= l.N && l.N <= r.End)
			}
		}
	}
	return v
}

// codes reads an item's excerpts. The nth codeStart is the first line of the
// nth code.
func (c *Client) codes(it *microdata.Item) []*Excerpt {
	starts := it.Props["codeStart"]
	var out []*Excerpt
	for i, v := range it.Props["code"] {
		ex := &Excerpt{}
		if i < len(starts) {
			ex.Start, _ = strconv.Atoi(starts[i].Text)
		}
		for j, text := range strings.Split(strings.TrimSuffix(microdata.TextContent(v.Node), "\n"), "\n") {
			ex.Lines = append(ex.Lines, Line{N: ex.Start + j, Text: text})
		}
		out = append(out, ex)
	}
	return out
}

// Place reads a symbol or file page.
func (c *Client) Place(ref string) (*Place, error) {
	p, err := c.Page(ref)
	if err != nil {
		return nil, err
	}
	it := p.Main
	pl := &Place{
		URL:    p.URL,
		Name:   it.Str("name"),
		Kind:   it.Str("kind"),
		File:   it.Str("file"),
		Symbol: it.Str("symbol"),
	}
	pl.StartLine, _ = strconv.Atoi(it.Str("startLine"))
	pl.EndLine, _ = strconv.Atoi(it.Str("endLine"))
	if codes := c.codes(it); len(codes) > 0 {
		pl.Code = codes[0]
	}
	if pl.Kind == "symbol" && pl.File != "" {
		pl.FileURL = c.Resolve("files/" + pl.File + "/")
		c.Ledger.Note("symbol-file-unlinked", p.URL, "%s in %s", pl.Name, pl.File)
		c.Ledger.Note("place-url-convention", p.URL, "guessed %s", pl.FileURL)
	}
	for _, r := range it.Items("anchoredBy") {
		pr := PlaceRef{
			URL:   r.Str("url"),
			Kind:  r.Str("kind"),
			Title: prose(r.Get("title")),
			Plan:  r.Str("plan"),
			Date:  r.Str("date"),
			Where: r.Str("where"),
			Added: refOf(r.Get("added")),
		}
		pr.New = r.Str("new") == "true"
		pl.AnchoredBy = append(pl.AnchoredBy, pr)
	}
	return pl, nil
}

// refOf reads a commit: a <data> with the full sha when it names a commit
// here, and otherwise the text as written ("none, a new repo").
func refOf(v *microdata.Value) Ref {
	if v == nil {
		return Ref{}
	}
	label := microdata.Squash(microdata.TextContent(v.Node))
	if v.Node.Data == "data" {
		return Ref{SHA: v.Text, Label: label}
	}
	return Ref{Label: label}
}

// prose renders a property's element as plain text: paragraphs separated by
// blank lines, code in backticks.
func prose(v *microdata.Value) string {
	if v == nil {
		return ""
	}
	var b strings.Builder
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		switch {
		case n.Type == html.TextNode:
			b.WriteString(n.Data)
			return
		case n.Type == html.ElementNode && n.Data == "code":
			b.WriteString("`" + microdata.TextContent(n) + "`")
			return
		case n.Type == html.ElementNode && n.Data == "input" && microdata.Attr(n, "type") == "checkbox":
			// A Select's answer: which options were chosen is only in checked.
			if hasAttr(n, "checked") {
				b.WriteString("[x] ")
			} else {
				b.WriteString("[ ] ")
			}
			return
		case n.Type == html.ElementNode && (n.Data == "p" || n.Data == "li" || n.Data == "pre"):
			if b.Len() > 0 {
				b.WriteString("\n\n")
			}
		}
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			walk(ch)
		}
	}
	walk(v.Node)
	paras := strings.Split(b.String(), "\n\n")
	out := paras[:0]
	for _, p := range paras {
		if p = microdata.Squash(p); p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, "\n\n")
}

// linkSentence is the sentence an answeredBy link sits in, which carries the
// verdict ("Answered in the as-built: held.").
func hasAttr(n *html.Node, key string) bool {
	for _, a := range n.Attr {
		if a.Key == key {
			return true
		}
	}
	return false
}

func hasCheckbox(n *html.Node) bool {
	if n.Type == html.ElementNode && n.Data == "input" && microdata.Attr(n, "type") == "checkbox" {
		return true
	}
	for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
		if hasCheckbox(ch) {
			return true
		}
	}
	return false
}

func linkSentence(v *microdata.Value) string {
	if p := v.Node.Parent; p != nil {
		return prose(&microdata.Value{Node: p})
	}
	return prose(v)
}

// facts reads the <dl class="facts"> as shown, for display only.
func facts(n *html.Node) [][2]string {
	var out [][2]string
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "dl" && microdata.HasClass(n, "facts") {
			var dt string
			for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
				if ch.Type != html.ElementNode {
					continue
				}
				switch ch.Data {
				case "dt":
					dt = microdata.Squash(microdata.TextContent(ch))
				case "dd":
					out = append(out, [2]string{dt, prose(&microdata.Value{Node: ch})})
				}
			}
			return
		}
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			walk(ch)
		}
	}
	walk(n)
	return out
}

func stripFragment(u string) string {
	if i := strings.IndexByte(u, '#'); i >= 0 {
		return u[:i]
	}
	return u
}

// Fragment is the part of u after #, or "".
func Fragment(u string) string {
	if i := strings.IndexByte(u, '#'); i >= 0 {
		return u[i+1:]
	}
	return ""
}

// Unfragment is u without its fragment.
func Unfragment(u string) string { return stripFragment(u) }
