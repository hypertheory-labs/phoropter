package substrate

import "strings"

// Site is every plan, as-built, and place reachable from the home page.
type Site struct {
	Home   *Home
	Docs   map[string]*Doc // by URL
	Places map[string]*Place
}

// Crawl reads the whole site, which fills the ledger with every gap the site
// has, not only the ones a session happened to walk past.
func (c *Client) Crawl() (*Site, error) {
	h, err := c.Home()
	if err != nil {
		return nil, err
	}
	s := &Site{Home: h, Docs: map[string]*Doc{}, Places: map[string]*Place{}}
	for _, p := range h.Plans {
		for _, u := range []string{p.URL, p.AsBuiltURL} {
			if u == "" {
				continue
			}
			d, err := c.Doc(u)
			if err != nil {
				return nil, err
			}
			s.Docs[d.URL] = d
		}
	}
	for _, p := range h.Places {
		pl, err := c.Place(p.URL)
		if err != nil {
			return nil, err
		}
		s.Places[pl.URL] = pl
	}
	for _, d := range s.Docs {
		for _, e := range d.Entries {
			if e.Kind != "Guess" || len(e.AnsweredBy) > 0 {
				continue
			}
			for _, by := range e.AnchoredBy {
				if other := s.Entry(by.URL); other != nil && other.Kind == "Guess" && len(other.AnsweredBy) > 0 {
					c.Ledger.Note("answer-elsewhere", d.URL+"#"+e.ID, "answered through %s", by.URL)
				}
			}
		}
	}
	return s, nil
}

// Entry finds the entry a URL with a fragment points at.
func (s *Site) Entry(u string) *Entry {
	d := s.Docs[Unfragment(u)]
	if d == nil {
		return nil
	}
	id := Fragment(u)
	for i := range d.Entries {
		if strings.EqualFold(d.Entries[i].ID, id) {
			return &d.Entries[i]
		}
	}
	return nil
}
