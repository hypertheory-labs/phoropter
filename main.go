// phoropter: read code through the plans that shaped it (organdy).
//
//	phoropter [path] [--at <commit>] [--port <n>]        serve the repository you're in
//	phoropter build [path] [--at <commit>] [--out site]  write the static site
//	phoropter index [path] [--at <commit>]               commit the indexes plans need
//	phoropter index --check <program or indexer dir>     check an indexer against indexers/README.md
//	phoropter symbols <file> [--scheme <scheme>]         list a file's symbols, for a plan's where:
//	phoropter url [path]                                 print where phoropter is serving this repository
//	phoropter ui [site]                                  read the site in the terminal
//	phoropter ui gaps [site]                             crawl it, and print where it read past the profile
//
// With no command it finds the repository's root, indexes what's missing,
// builds the site at HEAD, and serves it with the draft and notes in drafts/.
// "You run ph in the repo" (Jeff); everything else is secondary to that.
package main

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/hypertheory-labs/phoropter/internal/site"
	"github.com/hypertheory-labs/phoropter/tui/app"
)

//go:embed guide.md
var guideMarkdown string

const usage = `usage:
  phoropter [path] [--at <commit>] [--port <n>]        serve the repository you're in
  phoropter build [path] [--at <commit>] [--out site]  write the static site
  phoropter index [path] [--at <commit>]               commit the indexes plans need into .phoropter/index/
  phoropter index --check <program or indexer dir>     check an indexer against indexers/README.md
  phoropter symbols <file> [--scheme <scheme>]         list a file's symbols, ready for a plan's where:
  phoropter url [path]                                 print the address phoropter is serving this repository at
  phoropter ui [site]                                  read the site in the terminal (default: the one phoropter url finds)
  phoropter ui gaps [site]                             crawl the site and print where it read past the profile`

func main() {
	args := os.Args[1:]
	command := "serve"
	if len(args) > 0 {
		switch args[0] {
		case "build", "index", "url", "symbols", "ui", "help", "-h", "--help":
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
		served, err := site.Serve(site.ServeOptions{Repo: repo, Ref: *at, Out: dir, Port: *port, ExactPort: portSet, Store: store, Guide: guide})
		if err != nil {
			fail(err.Error())
		}
		file := servingFile(root, os.Getpid())
		os.MkdirAll(filepath.Dir(file), 0o755)
		info, _ := json.Marshal(map[string]any{"url": served.URL, "root": root, "commit": served.Site.Generated.SHA, "pid": os.Getpid(), "started": time.Now().UnixMilli()})
		os.WriteFile(file, append(info, '\n'), 0o644)
		fmt.Printf("%s at %s: %d pages\nserving %s\n", served.Site.RepoName, served.Site.Generated.Short, served.Pages, served.URL)
		remind(served.Site, store)
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
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
		fail(fmt.Sprintf("no indexer here handles %s files (see indexers/README.md)", ext))
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
	for _, info := range candidates {
		o, err := site.RunIndexer(info.Command, info.Cwd, tree, nil)
		if err != nil {
			fail(err.Error())
		}
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
		fail(fmt.Sprintf("nothing defined in %s at %s (uncommitted changes aren't indexed)", rel, repo.Short(sha)))
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
		if p, err := os.FindProcess(info.PID); err != nil || p.Signal(syscall.Signal(0)) != nil {
			os.Remove(filepath.Join(dir, name))
			continue
		}
		if info.Started > bestStarted {
			best, bestStarted = info.URL, info.Started
		}
	}
	return best
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
