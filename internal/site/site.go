package site

import (
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

// Builds the site model from one repo at one commit: its plans, their
// as-built files, each anchor shown at the entry's own commit and at the
// generated one, and the reverse pages ("places") the anchors point at.

type Commit struct {
	SHA, Short, Role string
	Time             int64 // for the commit that added a plan
}

type ResolvedAnchor struct {
	Anchor Anchor
	Target Target
	Place  *Place
	Entry  *EntryRef // an anchor at another plan's entry
	Plan   *Plan     // an anchor at a whole plan or as-built file
	Views  []*View
	Note   string // for anchors that are shown but not resolved
}

type EntryRef struct {
	Plan  *Plan
	Doc   *Doc
	Entry *ResolvedEntry
}

type ResolvedEntry struct {
	*Entry
	Resolved   []*ResolvedAnchor
	Answers    *EntryRef  // an Answered, to its plan's Guess of the same title
	AnsweredBy []EntryRef // a Guess, to the answers in its as-built
	AnchoredBy []EntryRef // entries in other plans whose anchors name this one
}

type Doc struct {
	File       string // repo-relative markdown path
	Href       string // site-relative directory, ending in /
	Doc        *PlanDoc
	Before     *Commit // base for a plan, head for an as-built
	BeforeText string  // as written, in case it isn't a commit
	Entries    []*ResolvedEntry
}

type Plan struct {
	Doc
	Name        string  // from the filename: 2026-10-29-pumice.md is pumice
	Date        string  // from the filename too, the only place it's written
	Added       *Commit // the commit that added the file; what orders plans
	AsBuilt     *Doc
	Revisits    *Plan // bodkin's revisits:, when it names a plan on this site
	RevisitedBy []*Plan
}

type PlaceRef struct {
	Plan   *Plan
	Doc    *Doc
	Entry  *ResolvedEntry
	Anchor Anchor
}

type Place struct {
	Key        string
	Kind       string // symbol or file
	Label      string // what the page is about, short
	Full       string // the symbol or path in full
	Href       string
	Definition *Definition // at the generated commit, for symbols
	Code       *Hunk       // the whole definition at the generated commit
	Refs       []PlaceRef
}

type Site struct {
	RepoName  string
	Generated Commit
	Plans     []*Plan
	Places    []*Place
	Context   *Context // what resolving one more document needs, for the draft
}

// Context is everything an anchor resolves against. Kept after the build so
// the dev server can resolve a draft per request without rebuilding the site.
type Context struct {
	Repo      *Repo
	RepoName  string
	Generated Commit
	Docs      []*Doc
	PlanOf    map[*Doc]*Plan
	OnSite    map[string]bool
	Places    map[string]*Place
	Store     *Store
}

func (cx *Context) IndexFor(c Commit, packages map[string]bool) *Lookup {
	if len(packages) == 0 {
		return nil
	}
	return cx.Store.Lookup(c.SHA, packages)
}

// localeCompare, as JavaScript's default collation orders strings.
var collator = collate.New(language.Und)

func localeLess(a, b string) bool { return collator.CompareString(a, b) < 0 }

var datePrefixRE = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-`)
var dateRE = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})-`)

func BuildSite(repo *Repo, ref string, store *Store) (*Site, error) {
	sha := repo.Resolve(ref)
	if sha == "" {
		return nil, fmt.Errorf("not a commit in %s: %s", repo.Dir, ref)
	}
	if store == nil {
		store = NewStore(repo, "", nil)
	}
	// The repository is named for its directory, resolved first, so "." and
	// "../.." name it as an absolute path would.
	abs, err := filepath.Abs(repo.Dir)
	if err != nil {
		return nil, err
	}
	repoName := filepath.Base(abs)
	generated := Commit{SHA: sha, Short: repo.Short(sha), Role: "generated"}

	var files []string
	isPlanFile := regexp.MustCompile(`^plans/[^/]+\.md$`)
	for _, f := range repo.Files(sha, "plans") {
		if isPlanFile.MatchString(f) {
			files = append(files, f)
		}
	}
	has := map[string]bool{}
	for _, f := range files {
		has[f] = true
	}

	var plans []*Plan
	for _, file := range files {
		if strings.HasSuffix(file, ".as-built.md") || strings.HasSuffix(file, ".notes.md") {
			continue
		}
		text, _ := repo.Show(sha, file)
		doc := ParsePlan(text)
		// The filename, not the cart: a cart is optional (lanyard), and plans
		// without one would share a page.
		name := datePrefixRE.ReplaceAllString(strings.TrimSuffix(path.Base(file), ".md"), "")
		href := "plans/" + Slugify(name) + "/"
		base, _ := doc.FM("base")
		_, hasBase := doc.FM("base")
		p := &Plan{Doc: Doc{File: file, Href: href, Doc: doc}, Name: name}
		p.Before, p.BeforeText = commitOf(repo, base, hasBase, "base")
		if m := dateRE.FindStringSubmatch(path.Base(file)); m != nil {
			p.Date = m[1]
		}
		if a, t, ok := repo.Added(sha, file); ok {
			p.Added = &Commit{SHA: a, Short: repo.Short(a), Role: "added", Time: t}
		}
		abFile := strings.TrimSuffix(file, ".md") + ".as-built.md"
		if has[abFile] {
			abText, _ := repo.Show(sha, abFile)
			abDoc := ParsePlan(abText)
			head, hasHead := abDoc.FM("head")
			ab := &Doc{File: abFile, Href: href + "as-built/", Doc: abDoc}
			ab.Before, ab.BeforeText = commitOf(repo, head, hasHead, "as-built head")
			p.AsBuilt = ab
		}
		plans = append(plans, p)
	}

	// Oldest first: by the commit that added each plan, since several plans can
	// share a date. The filename breaks a tie.
	sort.SliceStable(plans, func(i, j int) bool {
		ti, tj := addedTime(plans[i]), addedTime(plans[j])
		if ti != tj {
			return ti < tj
		}
		return localeLess(plans[i].File, plans[j].File)
	})

	for _, p := range plans {
		rv, _ := p.Doc.Doc.FM("revisits")
		for _, q := range plans {
			if q.File == rv {
				p.Revisits = q
				q.RevisitedBy = append(q.RevisitedBy, p)
				break
			}
		}
	}

	// Every entry exists before any anchor resolves, since an anchor can name
	// an entry in a plan that comes later.
	var docs []*Doc
	planOf := map[*Doc]*Plan{}
	for _, p := range plans {
		docs = append(docs, &p.Doc)
		planOf[&p.Doc] = p
		if p.AsBuilt != nil {
			docs = append(docs, p.AsBuilt)
			planOf[p.AsBuilt] = p
		}
	}
	onSite := map[string]bool{}
	for _, d := range docs {
		onSite[d.File] = true
		d.Entries = nil
		for _, e := range d.Doc.Entries {
			d.Entries = append(d.Entries, &ResolvedEntry{Entry: e})
		}
	}

	cx := &Context{Repo: repo, RepoName: repoName, Generated: generated, Docs: docs, PlanOf: planOf, OnSite: onSite, Places: map[string]*Place{}, Store: store}
	packages := PackagesOf(cx, docs)

	for _, p := range plans {
		ds := []*Doc{&p.Doc}
		if p.AsBuilt != nil {
			ds = append(ds, p.AsBuilt)
		}
		for _, d := range ds {
			// A plan's anchors show at its base, at its as-built's head (where its
			// own work landed, so lanyard's drift starts there), and at the
			// generated commit. An as-built's show at its head and the generated one.
			var head *Commit
			if d == &p.Doc && p.AsBuilt != nil {
				head = p.AsBuilt.Before
			}
			commits := []Commit{generated}
			if head != nil && head.SHA != sha {
				commits = append([]Commit{*head}, commits...)
			}
			if d.Before != nil && !hasSHA(commits, d.Before.SHA) {
				commits = append([]Commit{*d.Before}, commits...)
			}
			ResolveEntries(cx, p, d, commits, packages, true)
		}
	}

	// An as-built's Answered entry answers its plan's Guess with the exact title.
	for _, p := range plans {
		if p.AsBuilt == nil {
			continue
		}
		for _, answer := range p.AsBuilt.Entries {
			if answer.Kind != "Answered" {
				continue
			}
			var guess *ResolvedEntry
			for _, e := range p.Doc.Entries {
				if e.Kind == "Guess" && e.Title == answer.Title {
					guess = e
					break
				}
			}
			if guess == nil {
				continue
			}
			answer.Answers = &EntryRef{Plan: p, Doc: &p.Doc, Entry: guess}
			guess.AnsweredBy = append(guess.AnsweredBy, EntryRef{Plan: p, Doc: p.AsBuilt, Entry: answer})
		}
	}

	for _, place := range cx.Places {
		if place.Definition == nil {
			continue
		}
		def := place.Definition
		text, _ := repo.Show(sha, def.File)
		place.Code = &Hunk{Start: def.StartLine, Lines: slice(strings.Split(text, "\n"), def.StartLine-1, def.EndLine), Marked: []int{}}
	}

	// A reverse page lists newest first: someone arriving at a file usually
	// wants what touched it last.
	for _, place := range cx.Places {
		sort.SliceStable(place.Refs, func(i, j int) bool { return addedTime(place.Refs[i].Plan) > addedTime(place.Refs[j].Plan) })
	}

	var places []*Place
	for _, pl := range cx.Places {
		places = append(places, pl)
	}
	sort.SliceStable(places, func(i, j int) bool { return localeLess(places[i].Href, places[j].Href) })
	return &Site{RepoName: repoName, Generated: generated, Plans: plans, Places: places, Context: cx}, nil
}

func addedTime(p *Plan) int64 {
	if p.Added == nil {
		return 0
	}
	return p.Added.Time
}

func hasSHA(cs []Commit, sha string) bool {
	for _, c := range cs {
		if c.SHA == sha {
			return true
		}
	}
	return false
}

// PackagesOf is the package keys a document's symbol anchors need indexed.
func PackagesOf(cx *Context, docs []*Doc) map[string]bool {
	packages := map[string]bool{}
	for _, d := range docs {
		for _, e := range d.Doc.Entries {
			for _, a := range e.Anchors {
				t := TargetOf(a, cx.RepoName, cx.OnSite)
				if t.Kind == TSymbol && !a.IsNew {
					packages[PackageKey(t.Symbol)] = true
				}
			}
		}
	}
	return packages
}

// ResolveEntries resolves each anchor in d at commits. A registered document
// is part of the site: it's listed on the places it names and on the entries
// it anchors to. A draft isn't, so it only reads them, and links a place only
// if the site already has its page.
func ResolveEntries(cx *Context, p *Plan, d *Doc, commits []Commit, packages map[string]bool, register bool) {
	repo, generated := cx.Repo, cx.Generated
	for _, re := range d.Entries {
		re.Resolved = nil
		for _, anchor := range re.Anchors {
			t := TargetOf(anchor, cx.RepoName, cx.OnSite)
			ra := &ResolvedAnchor{Anchor: anchor, Target: t}
			re.Resolved = append(re.Resolved, ra)
			switch t.Kind {
			case TCrossRepo:
				ra.Note = fmt.Sprintf("in %s, which this site doesn't read", t.Repo)
				continue
			case TPlan:
				var doc *Doc
				for _, x := range cx.Docs {
					if x.File == t.File {
						doc = x
						break
					}
				}
				owner := cx.PlanOf[doc]
				where := owner.Name
				if doc != &owner.Doc {
					where = owner.Name + "'s as-built"
				}
				if !t.HasTitle {
					ra.Plan = owner
					if anchor.HasQuote && !anchor.IsNew {
						for _, c := range commits {
							ra.Views = append(ra.Views, ResolveAt(repo, Target{Kind: TFile, File: t.File}, anchor.Quote, true, c, nil))
						}
						ra.Views = Collapse(ra.Views)
					}
					continue
				}
				var entry *ResolvedEntry
				for _, e := range doc.Entries {
					if e.Title == t.Title {
						entry = e
						break
					}
				}
				if entry == nil {
					ra.Note = "no entry titled this in " + where
					continue
				}
				ra.Entry = &EntryRef{Plan: owner, Doc: doc, Entry: entry}
				if register {
					entry.AnchoredBy = append(entry.AnchoredBy, EntryRef{Plan: p, Doc: d, Entry: re})
				}
				if anchor.HasQuote {
					ra.Views = []*View{ResolveInEntry(repo, t.File, t.Title, anchor.Quote, generated)}
				}
				continue
			}
			if register {
				var defs map[string]Definition
				if l := cx.IndexFor(generated, packages); l != nil {
					defs = l.Definitions
				}
				ra.Place = placeFor(cx.Places, t, defs)
				ra.Place.Refs = append(ra.Place.Refs, PlaceRef{Plan: p, Doc: d, Entry: re, Anchor: anchor})
			} else {
				ra.Place = cx.Places[placeKey(t)]
			}
			if anchor.IsNew {
				ra.Note = "made by the work, so there is nothing to resolve"
				continue
			}
			if t.Kind == TFile && !anchor.HasQuote {
				ra.Note = "no quote, so nothing to search for"
				continue
			}
			for _, c := range commits {
				var idx *Lookup
				if t.Kind == TSymbol {
					idx = cx.IndexFor(c, packages)
				}
				ra.Views = append(ra.Views, ResolveAt(repo, t, anchor.Quote, anchor.HasQuote, c, idx))
			}
			ra.Views = Collapse(ra.Views)
		}
	}
}

func commitOf(repo *Repo, text string, has bool, role string) (*Commit, string) {
	if !has || text == "" {
		return nil, text
	}
	if sha := repo.Resolve(text); sha != "" {
		return &Commit{SHA: sha, Short: repo.Short(sha), Role: role}, text
	}
	return nil, text
}

func placeKey(t Target) string {
	switch t.Kind {
	case TSymbol:
		return "symbol " + t.Symbol
	case TFile:
		return "file " + t.File
	}
	return ""
}

var segmentRE = regexp.MustCompile("[#./`()\\[\\]]+")

func placeFor(places map[string]*Place, t Target, generatedIndex map[string]Definition) *Place {
	key := placeKey(t)
	if p, ok := places[key]; ok {
		return p
	}
	var p *Place
	if t.Kind == TSymbol {
		def, ok := generatedIndex[t.Symbol]
		descriptor := strings.Join(strings.Split(t.Symbol, " ")[4:], " ")
		name := descriptor
		if i := strings.LastIndex(descriptor, "`/"); i >= 0 {
			name = descriptor[i+2:]
		}
		if name == "" {
			name = descriptor
		}
		var segments []string
		for _, s := range segmentRE.Split(name, -1) {
			if s != "" {
				segments = append(segments, s)
			}
		}
		href := "symbols/not-at-generated/" + Slugify(t.Symbol) + "/"
		p = &Place{Key: key, Kind: "symbol", Label: name, Full: t.Symbol}
		if ok {
			href = "symbols/" + def.File + "/" + strings.Join(segments, "/") + "/"
			d := def
			p.Definition = &d
		}
		p.Href = href
	} else {
		p = &Place{Key: key, Kind: "file", Label: t.File, Full: t.File, Href: "files/" + t.File + "/"}
	}
	places[key] = p
	return p
}
