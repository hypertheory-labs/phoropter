package site

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Serve builds the site, then serves it read-only. A commit's site never
// changes, but HEAD does: served at HEAD, a request after HEAD moves builds the
// new commit first and swaps it in whole (poplin). The other exceptions are in
// the repo's working tree, under drafts/: the draft, read on every request to
// draft/ (loden), and linked from every page's header while it exists; and
// notes on landed plans, shown on each plan's page and counted on the home
// page (tarlatan, organza).

type ServeOptions struct {
	Repo      *Repo
	Ref       string
	Out       string
	Port      int
	ExactPort bool // otherwise a taken port moves to the next free one (batik)
	Store     *Store
	Guide     Guide
	OnRebuild func(s *Site, pages int) // after a rebuild is swapped in
}

type Served struct {
	Port   int
	URL    string
	Pages  int
	Site   *Site // as first built
	Server *http.Server
	live   *live
}

// Rebuild builds the site again now, at the ref it was served at.
func (s *Served) Rebuild() error {
	_, err := s.live.current(true)
	return err
}

// live is what changes while serving: the site and the directory it's in.
type live struct {
	mu      sync.Mutex
	o       ServeOptions
	site    *Site
	root    string
	checked time.Time
	failed  string // the commit a rebuild last failed at, so it isn't retried every second
	n       int
}

// current is the site to serve this request from. At HEAD, it looks at most
// once a second whether HEAD has moved; git rev-parse costs a few
// milliseconds. A rebuild goes into a fresh directory, so no request sees half
// a site, and a failed one leaves the last site up.
func (l *live) current(force bool) (*handler, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	var err error
	if force || (l.o.Ref == "HEAD" && time.Since(l.checked) >= time.Second) {
		l.checked = time.Now()
		sha := l.o.Repo.Resolve(l.o.Ref)
		if force || (sha != "" && sha != l.site.Generated.SHA && sha != l.failed) {
			if err = l.rebuild(); err != nil {
				l.failed = sha
				fmt.Fprintf(os.Stderr, "phoropter: couldn't rebuild at %s, still serving %s: %v\n", l.o.Repo.Short(sha), l.site.Generated.Short, err)
			}
		}
	}
	return &handler{site: l.site, root: l.root, repoDir: l.o.Repo.Dir}, err
}

func (l *live) rebuild() error {
	l.n++
	dir := fmt.Sprintf("%s.rebuild-%d", l.o.Out, l.n)
	s, pages, err := Build(l.o.Repo, l.o.Ref, dir, l.o.Store, l.o.Guide)
	if err != nil {
		os.RemoveAll(dir)
		return err
	}
	old := l.root
	l.site, l.root, l.failed = s, dir, ""
	os.RemoveAll(old)
	if l.o.OnRebuild != nil {
		l.o.OnRebuild(s, pages)
	}
	return nil
}

func (l *live) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h, _ := l.current(false)
	h.ServeHTTP(w, r)
}

func Serve(o ServeOptions) (*Served, error) {
	old, _ := filepath.Glob(o.Out + ".rebuild-*")
	for _, d := range old {
		os.RemoveAll(d)
	}
	s, pages, err := Build(o.Repo, o.Ref, o.Out, o.Store, o.Guide)
	if err != nil {
		return nil, err
	}
	h := &live{o: o, site: s, root: o.Out, checked: time.Now()}
	port := o.Port
	var ln net.Listener
	for {
		ln, err = net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err == nil {
			break
		}
		if o.ExactPort || !errors.Is(err, syscall.EADDRINUSE) || port >= o.Port+100 {
			return nil, err
		}
		port++
	}
	srv := &http.Server{Handler: h}
	go srv.Serve(ln)
	return &Served{Port: port, URL: fmt.Sprintf("http://localhost:%d/", port), Pages: pages, Site: s, Server: srv, live: h}, nil
}

// handler serves one request from one site.
type handler struct {
	site    *Site
	root    string
	repoDir string
}

var types = map[string]string{".html": "text/html; charset=utf-8", ".css": "text/css; charset=utf-8"}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	p := r.URL.Path
	if p == "/draft" {
		http.Redirect(w, r, "/draft/", http.StatusMovedPermanently)
		return
	}
	if p == "/draft/" {
		d, err := ReadDraft(h.site, h.repoDir)
		if err != nil {
			text(w, http.StatusInternalServerError, fmt.Sprintf("couldn't read %s: %v\n", DraftFile, err))
			return
		}
		if d == nil {
			text(w, http.StatusNotFound, fmt.Sprintf("no %s in %s\n", DraftFile, h.repoDir))
			return
		}
		send(w, r, "text/html; charset=utf-8", []byte(h.withDraftLink(DraftPage(h.site, d).HTML, "draft/")))
		return
	}
	if p == "/lenses" {
		http.Redirect(w, r, "/lenses/", http.StatusMovedPermanently)
		return
	}
	if strings.HasPrefix(p, "/lenses/") {
		h.serveLens(w, r, strings.Trim(strings.TrimPrefix(p, "/lenses/"), "/"))
		return
	}
	file := filepath.Join(h.root, filepath.FromSlash(path.Clean("/"+p)))
	root, _ := filepath.Abs(h.root)
	if abs, _ := filepath.Abs(file); !strings.HasPrefix(abs, root) {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	st, err := os.Stat(file)
	if err == nil && st.IsDir() {
		if !strings.HasSuffix(p, "/") {
			http.Redirect(w, r, p+"/", http.StatusMovedPermanently)
			return
		}
		file = filepath.Join(file, "index.html")
		_, err = os.Stat(file)
	}
	if err != nil {
		text(w, http.StatusNotFound, "not found\n")
		return
	}
	body, err := os.ReadFile(file)
	if err != nil {
		text(w, http.StatusNotFound, "not found\n")
		return
	}
	typ, ok := types[filepath.Ext(file)]
	if !ok {
		typ = "application/octet-stream"
	}
	if strings.HasPrefix(typ, "text/html") {
		rel := strings.TrimPrefix(p, "/")
		here := strings.TrimSuffix(rel, "index.html")
		body = []byte(h.withDraftLink(h.withPlanNotes(string(body), here), rel))
	}
	send(w, r, typ, body)
}

func text(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(code)
	w.Write([]byte(msg))
}

func send(w http.ResponseWriter, r *http.Request, typ string, body []byte) {
	w.Header().Set("Content-Type", typ)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		w.Write(body)
	}
}

// withPlanNotes: built pages don't know about notes on landed plans. On a
// plan's page, they go after the entries they quote, and notes on the whole
// plan before the first entry. On the home page, each plan with notes says
// how many.
func (h *handler) withPlanNotes(page, here string) string {
	if here == "" {
		for _, p := range h.site.Plans {
			notes, ok := ReadPlanNotes(h.repoDir, p)
			if !ok {
				continue
			}
			start := strings.Index(page, fmt.Sprintf(`itemprop="url" href="%s"`, p.Href))
			if start == -1 {
				continue
			}
			end := strings.Index(page[start:], "</span>\n</li>")
			if end == -1 {
				continue
			}
			end += start
			n := len(notes)
			word := fmt.Sprintf("%d notes", n)
			if n == 1 {
				word = "1 note"
			}
			page = page[:end] + fmt.Sprintf(`, <data itemprop="noteCount" value="%d">%s</data>`, n, word) + page[end:]
		}
		return page
	}
	var plan *Plan
	for _, p := range h.site.Plans {
		if p.Href == here {
			plan = p
		}
	}
	if plan == nil {
		return page
	}
	notes, ok := ReadPlanNotes(h.repoDir, plan)
	if !ok {
		return page
	}
	for _, e := range plan.Doc.Entries {
		var mine []string
		for _, a := range notes {
			if a.Found && a.At == e.Slug {
				mine = append(mine, AnnotationHTML(a))
			}
		}
		start := strings.Index(page, fmt.Sprintf(`<section class="entry" id="%s"`, e.Slug))
		if len(mine) == 0 || start == -1 {
			continue
		}
		end := strings.Index(page[start:], "</section>")
		if end == -1 {
			continue
		}
		at := start + end + len("</section>")
		page = page[:at] + "\n" + strings.Join(mine, "\n") + page[at:]
	}
	var top []string
	for _, a := range notes {
		if !a.HasQuote || !a.Found || a.At == "intro" {
			top = append(top, AnnotationHTML(a))
		}
	}
	if first := strings.Index(page, `<section class="entry"`); first != -1 {
		label := "<p class=\"note\">Notes on this plan, from <code>drafts/</code> in the working tree. The plan itself stays as written.</p>\n"
		page = page[:first] + label + strings.Join(top, "\n") + "\n" + page[first:]
	}
	return page
}

// serveLens: lenses are read from drafts/lenses/ on every request, like the
// draft (percale).
func (h *handler) serveLens(w http.ResponseWriter, r *http.Request, name string) {
	lenses := ReadLenses(h.site, h.repoDir)
	first := FirstCommit(&Repo{Dir: h.repoDir})
	if name == "" {
		send(w, r, "text/html; charset=utf-8", []byte(h.withDraftLink(LensesPage(h.site, lenses, first).HTML, "lenses/")))
		return
	}
	for _, l := range lenses {
		if l.Name == name {
			here := "lenses/" + name + "/"
			send(w, r, "text/html; charset=utf-8", []byte(h.withDraftLink(LensPage(h.site, l, first).HTML, here)))
			return
		}
	}
	text(w, http.StatusNotFound, fmt.Sprintf("no lens %s in %s\n", name, LensDir))
}

var homeLinkRE = regexp.MustCompile(`(<a class="home-link"[^>]*>home</a>)`)

// withDraftLink: built pages don't know about the draft or lenses; the server
// adds the links while they exist, relative to the page they're on.
func (h *handler) withDraftLink(page, here string) string {
	dir := here
	if !strings.HasSuffix(here, "/") && here != "" {
		dir = here[:strings.LastIndex(here, "/")+1]
	}
	var links []string
	if HasDraft(h.repoDir) {
		links = append(links, fmt.Sprintf(`<a rel="draft" href="%s">draft</a>`, Rel(dir, "draft/")))
	}
	// Always, so a reader can start a lens before there is one.
	links = append(links, fmt.Sprintf(`<a rel="lenses" href="%s">lenses</a>`, Rel(dir, "lenses/")))
	return homeLinkRE.ReplaceAllString(page, "$1\n"+strings.Join(links, "\n"))
}
