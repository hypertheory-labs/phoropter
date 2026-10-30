// Package substrate turns phoropter's pages into typed values, reading only
// what the profile defines where it can, and noting in a Ledger each place it
// can't.
package substrate

import (
	"fmt"
	"regexp"
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

// Doc is a plan page or an as-built page; they share a shape.
type Doc struct {
	URL         string
	IsAsBuilt   bool
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
	MatchLine                int // 0 when unknown
	SameAs                   string
	Code                     *Excerpt
	CodeFrom                 string // the commit the code was copied from, for sameAs
}

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
type Page struct {
	URL        string
	Doc        *microdata.Document
	Repository string
	Commit     Ref
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
			p.Commit = refOf(it.Get("commit"))
		case p.Main == nil && microdata.Within(it.Node, "main"):
			p.Main = it
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
	h := &Home{URL: p.URL, Repository: p.Repository, Commit: p.Commit}
	for _, it := range p.Main.Items("plan") {
		s := PlanSummary{
			URL:   it.Str("url"),
			Title: prose(it.Get("title")),
			Name:  it.Str("name"),
			Date:  it.Str("date"),
			Added: refOf(it.Get("added")),
			Base:  c.base(it.Get("base"), p.URL),
		}
		// Everything else on the line is prose and unmarked links.
		meta := microdata.Squash(microdata.TextContent(it.Node))
		if m := entriesRE.FindStringSubmatch(meta); m != nil {
			s.Entries, _ = strconv.Atoi(m[1])
			c.Ledger.Note("home-entries-prose", p.URL, "%q", m[0])
		}
		for _, a := range anchorsIn(it.Node) {
			text := microdata.Squash(microdata.TextContent(a))
			href := c.Resolve(microdata.Attr(a, "href"))
			if microdata.Attr(a, "rel") != "" {
				continue
			}
			switch {
			case strings.Contains(text, "as-built"):
				s.AsBuiltURL = href
				c.Ledger.Note("home-as-built-unmarked", p.URL, "link text %q", text)
			case strings.Contains(precedingText(a), "revisits"):
				s.RevisitsURL = href
				c.Ledger.Note("home-revisits-unmarked", p.URL, "link text %q", text)
			}
		}
		h.Plans = append(h.Plans, s)
	}
	for _, it := range p.Main.Items("place") {
		s := PlaceSummary{URL: it.Str("url"), Name: it.Str("name"), Kind: it.Str("kind")}
		if m := entriesRE.FindStringSubmatch(microdata.Squash(microdata.TextContent(it.Node))); m != nil {
			s.Entries, _ = strconv.Atoi(m[1])
		}
		h.Places = append(h.Places, s)
	}
	return h, nil
}

var entriesRE = regexp.MustCompile(`(\d+) entr(?:y|ies)`)

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
		d.Base = c.base(v, p.URL)
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
	return d, nil
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

var lineRE = regexp.MustCompile(`quote at line (\d+)`)

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
	if code := it.Get("code"); code != nil {
		v.Code = c.excerpt(code.Node, page)
		for _, l := range v.Code.Lines {
			if l.Marked {
				v.MatchLine = l.N
				break
			}
		}
	}
	if m := lineRE.FindStringSubmatch(v.Note); m != nil {
		if n, _ := strconv.Atoi(m[1]); v.MatchLine == 0 || v.MatchLine == n {
			v.MatchLine = n
		}
		c.Ledger.Note("match-line-prose", page, "note %q", v.Note)
	}
	return v
}

// excerpt reads a <pre itemprop="code">: each line is a span.ln with its
// number, then the text, and the matched lines are wrapped in <mark>.
func (c *Client) excerpt(pre *html.Node, page string) *Excerpt {
	ex := &Excerpt{}
	ex.Start, _ = strconv.Atoi(microdata.Attr(pre, "data-start"))
	var walk func(n *html.Node, inLn, inMark bool)
	walk = func(n *html.Node, inLn, inMark bool) {
		if n.Type == html.ElementNode {
			inLn = inLn || microdata.HasClass(n, "ln")
			inMark = inMark || n.Data == "mark"
		}
		if n.Type == html.TextNode {
			if inLn {
				num, _ := strconv.Atoi(strings.TrimSpace(n.Data))
				ex.Lines = append(ex.Lines, Line{N: num, Marked: inMark})
			} else if len(ex.Lines) > 0 {
				last := &ex.Lines[len(ex.Lines)-1]
				// The newline between lines sits outside the line's own text.
				last.Text += strings.SplitN(n.Data, "\n", 2)[0]
			}
		}
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			walk(ch, inLn, inMark)
		}
	}
	walk(pre, false, false)
	c.Ledger.Note("line-numbers-unprofiled", page, "")
	return ex
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
	if v := it.Get("code"); v != nil {
		pl.Code = c.excerpt(v.Node, p.URL)
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
		if v := r.Get("new"); v != nil {
			pr.New = true
			if v.Node.Data != "meta" {
				c.Ledger.Note("new-two-shapes", p.URL, "<%s>%s</%s>", v.Node.Data, microdata.Squash(v.Text), v.Node.Data)
			}
		}
		pl.AnchoredBy = append(pl.AnchoredBy, pr)
	}
	return pl, nil
}

func (c *Client) base(v *microdata.Value, page string) Ref {
	if v == nil {
		return Ref{}
	}
	if v.Node.Data != "data" {
		c.Ledger.Note("base-not-commit", page, "%q", microdata.Squash(v.Text))
		return Ref{Label: microdata.Squash(v.Text)}
	}
	return refOf(v)
}

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

func anchorsIn(n *html.Node) []*html.Node {
	var out []*html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			out = append(out, n)
		}
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			walk(ch)
		}
	}
	walk(n)
	return out
}

func precedingText(n *html.Node) string {
	if s := n.PrevSibling; s != nil && s.Type == html.TextNode {
		return s.Data
	}
	return ""
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
