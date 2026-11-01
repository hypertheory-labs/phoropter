package substrate

import "strconv"

// Lenses (percale): a reader's own subsets of plans, which the dev server
// serves at lenses/ from drafts/lenses/. Read like any other page: by the
// profile's properties, never by the files.

type Lenses struct {
	URL         string
	FirstCommit string // the repository's root commit, which a lens's repository names
	Lenses      []LensSummary
}

type LensSummary struct {
	URL, Title, File string
	Items            int
}

type Lens struct {
	URL, Title, File, Intro    string
	Repository, RepositoryNote string
	Items                      []LensItem
}

type LensItem struct {
	URL, Kind, Title, Plan, Status string // Status: a guess's "open" or verdict
	Reference, Note, Problem       string
}

// IsLenses and IsLens say what a page's main item is, for choosing a screen.
func IsLenses(p *Page) bool { return p.Main.Get("firstCommit") != nil }
func IsLens(p *Page) bool   { return p.Main.Get("itemCount") != nil && p.Main.Get("entry") == nil }

func (c *Client) Lenses(ref string) (*Lenses, error) {
	p, err := c.Page(ref)
	if err != nil {
		return nil, err
	}
	ls := &Lenses{URL: p.URL, FirstCommit: p.Main.Str("firstCommit")}
	for _, it := range p.Main.Items("lens") {
		n, _ := strconv.Atoi(it.Str("itemCount"))
		ls.Lenses = append(ls.Lenses, LensSummary{URL: it.Str("url"), Title: prose(it.Get("title")), File: it.Str("file"), Items: n})
	}
	return ls, nil
}

func (c *Client) Lens(ref string) (*Lens, error) {
	p, err := c.Page(ref)
	if err != nil {
		return nil, err
	}
	m := p.Main
	l := &Lens{URL: p.URL, Title: prose(m.Get("title")), File: m.Str("file"), Intro: prose(m.Get("intro")),
		Repository: m.Str("repository"), RepositoryNote: m.Str("repositoryNote")}
	for _, it := range m.Items("lensItem") {
		l.Items = append(l.Items, LensItem{
			URL: it.Str("entry"), Kind: it.Str("kind"), Title: prose(it.Get("title")), Plan: it.Str("plan"),
			Status: it.Str("status"), Reference: it.Str("reference"), Note: prose(it.Get("note")), Problem: it.Str("problem"),
		})
	}
	return l, nil
}
