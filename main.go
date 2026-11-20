// phoropter: read code through the plans that shaped it (organdy).
//
//	phoropter [path] [--at <commit>] [--port <n>]        serve the repository you're in
//	phoropter init [path]                                write the first plan, how we work here, as a draft
//	phoropter doctor [path]                              say what this machine lacks for the repository, and how to fix it
//	phoropter build [path] [--at <commit>] [--out site]  write the static site
//	phoropter index [path] [--at <commit>]               commit the indexes plans need
//	phoropter index --check <program or indexer dir>     check an indexer against indexers/README.md
//	phoropter symbols <file> [--scheme <scheme>]         list a file's symbols, for a plan's where:
//	phoropter url [path]                                 print where phoropter is serving this repository
//	phoropter indexers                                   list the indexers this machine has
//	phoropter guide                                      print the plan format's guide
//	phoropter ui [site]                                  read the site in the terminal
//	phoropter ui gaps [site]                             crawl it, and print where it read past the profile
//
// With no command it finds the repository's root, indexes what's missing,
// builds the site at HEAD, and serves it with the draft and notes in drafts/,
// following HEAD as it moves (poplin).
// "You run ph in the repo" (Jeff); everything else is secondary to that.
package main

import (
	"bufio"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	neturl "net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/charmbracelet/x/term"
	"github.com/hypertheory-labs/phoropter/internal/site"
	"github.com/hypertheory-labs/phoropter/tui/app"
)

//go:embed guide.md
var guideMarkdown string

const usage = `usage:
  phoropter [path] [--at <commit>] [--port <n>]        serve the repository you're in
  phoropter init [path]                                write this repository's first plan, how you work here, as a draft
  phoropter doctor [path]                              say what this machine lacks for this repository, and the command for each
  phoropter build [path] [--at <commit>] [--out site]  write the static site
  phoropter index [path] [--at <commit>]               commit the indexes plans need into .phoropter/index/
  phoropter index --check <program or indexer dir>     check an indexer against indexers/README.md
  phoropter symbols <file> [--scheme <scheme>]         list a file's symbols, ready for a plan's where:
  phoropter url [path]                                 print the address phoropter is serving this repository at
  phoropter indexers                                   list the indexers this machine has, and where they were found
  phoropter guide                                      print the guide to writing plans (no server needed)
  phoropter ui [site]                                  read the site in the terminal (default: the one phoropter url finds)
  phoropter ui gaps [site]                             crawl the site and print where it read past the profile`

func main() {
	args := os.Args[1:]
	command := "serve"
	if len(args) > 0 {
		switch args[0] {
		case "init", "doctor", "build", "index", "url", "symbols", "ui", "indexers", "guide", "help", "-h", "--help":
			command, args = args[0], args[1:]
		}
	}
	if command == "help" || command == "-h" || command == "--help" {
		fmt.Println(usage)
		return
	}
	if command == "ui" {
		runUI(args)
		return
	}
	if command == "guide" {
		fmt.Print(guideMarkdown)
		return
	}
	if command == "indexers" {
		indexers()
		return
	}

	fs := flag.NewFlagSet("phoropter", flag.ExitOnError)
	at := fs.String("at", "HEAD", "commit")
	port := fs.Int("port", 4173, "port")
	out := fs.String("out", "", "output directory")
	check := fs.String("check", "", "indexer to check")
	scheme := fs.String("scheme", "", "scheme for symbols")
	fs.Usage = func() { fmt.Fprintln(os.Stderr, usage) }
	positionals := parseInterleaved(fs, args)
	portSet := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "port" {
			portSet = true
		}
	})

	var target string
	if command == "symbols" {
		if len(positionals) == 0 {
			fail("symbols needs a file")
		}
		target, positionals = positionals[0], positionals[1:]
	}
	start, _ := os.Getwd()
	if len(positionals) > 0 {
		start = positionals[0]
	} else if target != "" {
		start = filepath.Dir(abs(target))
	}
	root := repoRoot(start)
	if root == "" {
		fail(fmt.Sprintf("it reads a git repository, and %s isn't in one", abs(start)))
	}
	if command == "doctor" {
		repo := &site.Repo{Dir: root}
		facts := doctorFacts{git: gitVersion(), firstParty: site.FirstParty, indexers: site.Indexers(),
			unavailable: site.IndexersUnavailable(), schemes: map[string]int{}}
		if sha := repo.Resolve("HEAD"); sha != "" {
			facts.schemes = schemesInPlans(repo, sha)
		}
		out, missing := facts.report()
		fmt.Print(out)
		if missing {
			os.Exit(1)
		}
		return
	}
	if command == "url" {
		url := servingURL(root)
		if url == "" {
			fail(fmt.Sprintf("it isn't serving %s; run phoropter there", root))
		}
		fmt.Println(url)
		return
	}
	repo := &site.Repo{Dir: root}
	if repo.Resolve(*at) == "" {
		if *at == "HEAD" {
			fail(root + " has no commits yet; phoropter reads a repository at a commit")
		}
		fail(fmt.Sprintf("%s isn't a commit in %s", *at, root))
	}
	if command == "init" {
		base := repo.Short(repo.Resolve("HEAD"))
		saw := look(root)
		if err := writeFirstDraft(root, base, saw.isNew()); err == errDraftExists {
			fail("there's already a draft at " + draftPath + ", and a repository has one. Convert or delete it, then run init again.")
		} else if err != nil {
			fail(err.Error())
		}
		fmt.Print(initWalkthrough(root, servingURL(root), countPlans(root), saw))
		return
	}
	store := site.NewStore(repo, "", nil)
	guide := site.Guide{Markdown: guideMarkdown, Version: version()}
	short := func(p string) string {
		cwd, _ := os.Getwd()
		if r, err := filepath.Rel(cwd, p); err == nil {
			return r
		}
		return p
	}

	switch {
	case command == "symbols":
		symbols(repo, root, target, *at, *scheme)
	case command == "index" && *check != "":
		// A directory with an indexer.json (indexers/go, or the one you're
		// writing), or a command.
		dir := abs(*check)
		command, cwd := strings.Fields(*check), ""
		if b, err := os.ReadFile(filepath.Join(dir, "indexer.json")); err == nil {
			var m struct {
				Command []string `json:"command"`
			}
			json.Unmarshal(b, &m)
			command, cwd = m.Command, dir
		}
		ok, report := site.CheckIndexer(command, cwd, repo)
		fmt.Println(strings.Join(report, "\n"))
		if !ok {
			os.Exit(1)
		}
	case command == "index":
		s, err := site.BuildSite(repo, *at, store)
		if err != nil {
			fail(err.Error())
		}
		var written []string
		for _, pair := range uncommitted(s, store) {
			if p := store.Commit(pair[0], pair[1]); p != "" {
				written = append(written, "  "+short(p))
			}
		}
		if len(written) == 0 {
			fmt.Println("Nothing to write: every index the plans need is already in .phoropter/index/.")
		} else {
			fmt.Printf("Wrote %d to commit:\n%s\n", len(written), strings.Join(written, "\n"))
		}
	case command == "build":
		dir := *out
		if dir == "" {
			dir = "site"
		}
		began := time.Now()
		s, pages, err := site.Build(repo, *at, dir, store, guide)
		if err != nil {
			fail(err.Error())
		}
		fmt.Printf("%s at %s: %d pages in %s/ (%d ms)\n", s.RepoName, s.Generated.Short, pages, short(abs(dir)), time.Since(began).Milliseconds())
		remind(s, store)
	default:
		// serve: the site goes in the cache, never into the repository.
		dir := *out
		if dir == "" {
			dir = filepath.Join(site.CacheDir(), "sites", id(root))
		}
		file := servingFile(root, os.Getpid())
		started := time.Now().UnixMilli()
		var served *site.Served
		record := func(commit string) {
			os.MkdirAll(filepath.Dir(file), 0o755)
			info, _ := json.Marshal(map[string]any{"url": served.URL, "root": root, "commit": commit, "pid": os.Getpid(), "started": started})
			os.WriteFile(file, append(info, '\n'), 0o644)
		}
		served, err := site.Serve(site.ServeOptions{Repo: repo, Ref: *at, Out: dir, Port: *port, ExactPort: portSet, Store: store, Guide: guide,
			OnRebuild: func(s *site.Site, pages int) {
				record(s.Generated.SHA)
				fmt.Printf("rebuilt at %s: %d pages\n", s.Generated.Short, pages)
				remind(s, store)
			}})
		if err != nil {
			fail(err.Error())
		}
		record(served.Site.Generated.SHA)
		fmt.Printf("%s at %s: %d pages\nserving %s\n", served.Site.RepoName, served.Site.Generated.Short, served.Pages, served.URL)
		remind(served.Site, store)
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
		// The menu is for a person at a terminal. Run in the background, as an
		// agent usually runs it, nothing reads stdin (poplin).
		if term.IsTerminal(os.Stdin.Fd()) {
			fmt.Println("press h + enter to show help")
			go menu(served, stop)
		}
		<-stop
		os.Remove(file)
	}
}

// runUI is the TUI, against the site given, or the server phoropter url finds
// for the repository here, or phoropter's default address.
func runUI(args []string) {
	address := "http://localhost:4173/"
	if cwd, err := os.Getwd(); err == nil {
		if root := repoRoot(cwd); root != "" {
			if url := servingURL(root); url != "" {
				address = url
			}
		}
	}
	if err := app.Run(args, address); err != nil {
		fmt.Fprintln(os.Stderr, "phoropter ui:", err)
		os.Exit(1)
	}
}

func symbols(repo *site.Repo, root, target, at, scheme string) {
	rel, _ := filepath.Rel(root, abs(target))
	rel = filepath.ToSlash(rel)
	ext := filepath.Ext(rel)
	var all, candidates []site.IndexerInfo
	for _, i := range site.Indexers() {
		for _, e := range i.Extensions {
			if e == ext {
				all = append(all, i)
			}
		}
	}
	if len(all) == 0 {
		fail(noIndexerFor(ext))
	}
	for _, i := range all {
		if scheme == "" || i.Scheme == scheme {
			candidates = append(candidates, i)
		}
	}
	if len(candidates) == 0 {
		var schemes []string
		for _, i := range all {
			schemes = append(schemes, i.Scheme)
		}
		fail(fmt.Sprintf("no indexer for %s files has scheme %s; there's %s", ext, scheme, strings.Join(schemes, ", ")))
	}
	// An indexer that says what it needs won't run without it; say so here,
	// not by way of doctor, which only knows what plans anchor (cambric).
	for _, i := range candidates {
		if len(i.Needs) > 0 {
			var lines []string
			for _, n := range i.Needs {
				lines = append(lines, fmt.Sprintf("  %s: %s", n.What, n.Install))
			}
			fail(fmt.Sprintf("%s indexes %s files, and needs first:\n%s", i.Indexer, ext, strings.Join(lines, "\n")))
		}
	}
	sha := repo.Resolve(at)
	tree, err := repo.Checkout(sha)
	if err != nil {
		fail(err.Error())
	}
	defer site.RemoveDir(tree)
	// Every indexer for the file's extension, or the one --scheme names. With
	// more than one, each scheme's symbols are under their own heading (dimity).
	type sym struct {
		start, end int
		symbol     string
	}
	var headings []string
	byScheme := map[string][]sym{}
	var outputs []site.IndexerOutput
	for _, info := range candidates {
		o, err := site.RunIndexer(info.Command, info.Cwd, tree, nil)
		if err != nil {
			fail(err.Error() + "\nphoropter doctor says what this repository needs.")
		}
		outputs = append(outputs, o)
		var found []sym
		for _, p := range o.Packages {
			for symbol, v := range p.Definitions {
				f, _ := v[0].(string)
				s, _ := v[1].(float64)
				e, _ := v[2].(float64)
				if f == rel {
					found = append(found, sym{int(s), int(e), symbol})
				}
			}
		}
		if len(found) > 0 {
			h := fmt.Sprintf("%s (%s)", info.Scheme, info.Indexer)
			headings = append(headings, h)
			byScheme[h] = found
		}
	}
	if len(headings) == 0 {
		var indexed, empty []string
		for _, o := range outputs {
			if len(o.Packages) == 0 {
				empty = append(empty, o.Scheme)
			}
			for _, p := range o.Packages {
				for _, v := range p.Definitions {
					if f, ok := v[0].(string); ok {
						indexed = append(indexed, f)
					}
				}
			}
		}
		fail(nothingDefined(repo, rel, at, sha, indexed, empty))
	}
	for _, h := range headings {
		found := byScheme[h]
		sort.Slice(found, func(i, j int) bool {
			if found[i].start != found[j].start {
				return found[i].start < found[j].start
			}
			return found[i].symbol < found[j].symbol
		})
		indent := ""
		if len(headings) > 1 {
			fmt.Println(h + ":")
			indent = "  "
		}
		for _, f := range found {
			fmt.Printf("%s%-10s %s\n", indent, fmt.Sprintf("%d–%d", f.start, f.end), f.symbol)
		}
	}
}

// nothingDefined says why a file has no symbols, as far as phoropter can tell
// (poplin): changes that aren't committed, a copy elsewhere whose package has
// the same name and was indexed instead, or no named declarations at all.
func nothingDefined(repo *site.Repo, rel, at, sha string, indexed, empty []string) string {
	msg := fmt.Sprintf("nothing defined in %s at %s: ", rel, repo.Short(sha))
	if at == "HEAD" {
		if out, _ := exec.Command("git", "-C", repo.Dir, "status", "--porcelain", "--", rel).Output(); len(out) > 0 {
			return msg + "it has changes that aren't committed, and only commits are indexed"
		}
	}
	// An indexer that found no packages at all says nothing about this file's
	// declarations; say what it looks for instead (cambric).
	if len(indexed) == 0 && len(empty) > 0 {
		what := "the packages its indexer reads"
		if fp, ok := firstPartySchemes[empty[0]]; ok {
			what = fp.pkg
		}
		return msg + "the " + empty[0] + " indexer found no packages in this repository. It indexes packages: " + what +
			". Until there is one, a file anchor with a quote reaches this file"
	}
	if c := copyOf(rel, indexed); c != "" {
		return msg + c + " is indexed instead. It looks like a copy, and when copies' packages share a name, only the first path is indexed (see indexers/README.md); a file anchor with a quote reaches this one"
	}
	return msg + "it has no named declarations (top-level statements, say); a file anchor with a quote reaches it"
}

// copyOf is the indexed file that shares the longest tail of path with rel,
// at least a directory and a file name: …/Vendors/Api.cs for
// student/…/Vendors/Api.cs. It still matches after copies diverge.
func copyOf(rel string, indexed []string) string {
	parts := strings.Split(rel, "/")
	best, bestN := "", 1
	for _, f := range indexed {
		if f == rel {
			continue
		}
		fp := strings.Split(f, "/")
		n := 0
		for n < len(parts) && n < len(fp) && parts[len(parts)-1-n] == fp[len(fp)-1-n] {
			n++
		}
		if n > bestN || (n == bestN && n > 1 && f < best) {
			best, bestN = f, n
		}
	}
	return best
}

// indexers lists every indexer this machine has: its scheme, the extensions it
// indexes, its name and version, and where it was found (poplin).
func indexers() {
	all := site.Indexers()
	if site.FirstParty == "" {
		fmt.Println("first-party indexers: not found. They're in indexers/ in a clone of phoropter, found from a binary built there, or from PHOROPTER_INDEXERS.")
	}
	if len(all) == 0 {
		fmt.Println("No indexers here. File anchors with a quote work in any language; for symbols, the README's Indexers section says what to install, and indexers/README.md how to write one.")
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "SCHEME\tEXTENSIONS\tINDEXER\tFOUND")
	for _, i := range all {
		found := i.Cwd
		if found == "" {
			found = i.Command[0]
		}
		if r, err := filepath.Rel(filepath.Dir(site.FirstParty), found); site.FirstParty != "" && i.Cwd != "" && err == nil {
			found = r
		}
		fmt.Fprintf(w, "%s\t%s\t%s %s\t%s\n", i.Scheme, strings.Join(i.Extensions, " "), i.Indexer, i.Version, found)
	}
	w.Flush()
}

const menuHelp = `  r + enter   rebuild now
  o + enter   open the site in the browser
  u + enter   print the address
  q + enter   quit
  h + enter   show this`

// menu reads a key and enter at a time, as Vite does. Keys alone would need
// the terminal in raw mode (poplin).
func menu(served *site.Served, stop chan<- os.Signal) {
	in := bufio.NewScanner(os.Stdin)
	for in.Scan() {
		switch strings.TrimSpace(in.Text()) {
		case "r":
			served.Rebuild()
		case "o":
			open := map[string][]string{"darwin": {"open"}, "windows": {"rundll32", "url.dll,FileProtocolHandler"}}[runtime.GOOS]
			if open == nil {
				open = []string{"xdg-open"}
			}
			if err := exec.Command(open[0], append(open[1:], served.URL)...).Start(); err != nil {
				fmt.Fprintf(os.Stderr, "phoropter: couldn't open a browser: %v\n", err)
			}
		case "u":
			fmt.Println(served.URL)
		case "q":
			stop <- os.Interrupt
			return
		case "h":
			fmt.Println(menuHelp)
		case "":
		default:
			fmt.Println("press h + enter to show help")
		}
	}
}

// uncommitted is the plan commits whose indexes a build needed but found only
// in the cache.
func uncommitted(s *site.Site, store *site.Store) [][2]string {
	planCommits := map[string]bool{}
	for _, p := range s.Plans {
		if p.Before != nil {
			planCommits[p.Before.SHA] = true
		}
		if p.AsBuilt != nil && p.AsBuilt.Before != nil {
			planCommits[p.AsBuilt.Before.SHA] = true
		}
	}
	var out [][2]string
	for _, sha := range store.AskedSHAs {
		if !planCommits[sha] {
			continue
		}
		var keys []string
		for k := range store.Asked[sha] {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if !store.IsCommitted(sha, k) {
				out = append(out, [2]string{sha, k})
			}
		}
	}
	return out
}

func remind(s *site.Site, store *site.Store) {
	n := len(uncommitted(s, store))
	switch {
	case n == 1:
		fmt.Println("1 index that plans need is only in this machine's cache; phoropter index commits it.")
	case n > 1:
		fmt.Printf("%d indexes that plans need are only in this machine's cache; phoropter index commits them.\n", n)
	}
}

// repoRoot is the repository holding a path: its root, found by walking up
// to .git.
func repoRoot(start string) string {
	dir := abs(start)
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func id(root string) string {
	sum := sha256.Sum256([]byte(root))
	return hex.EncodeToString(sum[:])[:16]
}

// servingFile is where phoropter url finds a running server: a small file
// outside the repository, one per server, keyed by the repository's root and
// the server's process, removed when that server stops. One file per server,
// because two can serve one repository, and the first to stop mustn't erase
// the other.
func servingFile(root string, pid int) string {
	return filepath.Join(site.CacheDir(), "serving", fmt.Sprintf("%s.%d.json", id(root), pid))
}

// servingURL is the newest live server for a repository. Files left by
// servers that died without tidying up are removed on the way.
func servingURL(root string) string {
	dir := filepath.Join(site.CacheDir(), "serving")
	entries, _ := os.ReadDir(dir)
	best, bestStarted := "", int64(-1)
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, id(root)+".") || !strings.HasSuffix(name, ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		var info struct {
			URL     string `json:"url"`
			PID     int    `json:"pid"`
			Started int64  `json:"started"`
		}
		json.Unmarshal(b, &info)
		if !answers(info.URL) {
			os.Remove(filepath.Join(dir, name))
			continue
		}
		if info.Started > bestStarted {
			best, bestStarted = info.URL, info.Started
		}
	}
	return best
}

// answers says whether a server is listening at url's port (chiffon). A
// connection, not a request, so it can't start a rebuild, and it works where
// there's no signal 0 to send a process, as on Windows.
func answers(url string) bool {
	u, err := neturl.Parse(url)
	if err != nil || u.Port() == "" {
		return false
	}
	c, err := net.DialTimeout("tcp", "127.0.0.1:"+u.Port(), time.Second)
	if err != nil {
		return false
	}
	c.Close()
	return true
}

// noIndexerFor says how to get an indexer for a file extension, rather than
// sending you to doctor, which only knows what plans already anchor (cambric).
func noIndexerFor(ext string) string {
	msg := fmt.Sprintf("no indexer here handles %s files", ext)
	s := firstPartyFor(ext)
	if s == "" {
		return msg + ". phoropter ships none for it; indexers/README.md in phoropter's repository says how to write one, " +
			"and a file anchor with a quote works in any language"
	}
	d := doctorFacts{firstParty: site.FirstParty, indexers: site.Indexers(), unavailable: site.IndexersUnavailable()}
	return msg + ". To get one: " + d.missingFix(s)
}

// version is the binary's commit, from the build's own vcs stamp, saying so
// when it was built with uncommitted changes; otherwise its module version.
func version() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "an unknown version"
	}
	rev, dirty := "", false
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}
	if rev != "" {
		if len(rev) > 7 {
			rev = rev[:7]
		}
		if dirty {
			return "commit " + rev + ", with uncommitted changes"
		}
		return "commit " + rev
	}
	return "version " + info.Main.Version
}

// parseInterleaved lets flags come before or after positionals, as Node's
// parseArgs did.
func parseInterleaved(fs *flag.FlagSet, args []string) []string {
	var positionals []string
	for len(args) > 0 {
		fs.Parse(args)
		args = fs.Args()
		if len(args) > 0 {
			positionals = append(positionals, args[0])
			args = args[1:]
		}
	}
	return positionals
}

func abs(p string) string {
	a, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	return a
}

func fail(message string) {
	fmt.Fprintln(os.Stderr, "phoropter:", message)
	os.Exit(1)
}
