package site

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// The index store (batik). An index is a fact about one commit: in this
// package, at this sha, these symbols are defined on these lines. It never
// changes, so it's kept rather than recomputed.
//
//   - The indexes plans need (their base and as-built commits) are committed in
//     .phoropter/index/<sha>/<package>.json, so a plan's symbol anchors outlive
//     the toolchain that could index them. Only `phoropter index` writes there;
//     a build never writes into the repository it reads.
//   - Everything else, the generated commit's index above all, goes in a local
//     cache under $XDG_CACHE_HOME/phoropter (or ~/.cache/phoropter).
//
// A file holds only what phoropter reads, sorted, and not its own commit, so
// an unchanged package gives byte-identical files across commits and git
// stores them once. It names its scheme and indexer, so two schemes naming the
// same package read as missing rather than misread (seersucker). Packages are
// keyed `<scheme> <package>`, as PackageKey gives them.

const StoreDir = ".phoropter/index"

type stored struct {
	Indexer     string                    `json:"indexer"`
	Version     string                    `json:"version"`
	Scheme      string                    `json:"scheme,omitempty"`
	Package     string                    `json:"package"`
	Definitions map[string][3]interface{} `json:"definitions"`
}

func CacheDir() string {
	base := os.Getenv("XDG_CACHE_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".cache")
	}
	return filepath.Join(base, "phoropter")
}

// fileOf is a package name as a file name: @scope/name becomes @scope+name.
func fileOf(pkg string) string { return strings.ReplaceAll(pkg, "/", "+") + ".json" }

func splitKey(key string) (string, string) {
	i := strings.Index(key, " ")
	return key[:i], key[i+1:]
}

// Serialize writes one package's index: keys sorted, as the TypeScript
// version wrote it, so files written by either read the same.
func Serialize(found Found, pkg string, defs map[string]Definition) string {
	keys := make([]string, 0, len(defs))
	for k := range defs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	fmt.Fprintf(&b, "{\n \"indexer\": %s,\n \"version\": %s,\n \"scheme\": %s,\n \"package\": %s,\n \"definitions\": {", jsonString(found.Indexer), jsonString(found.Version), jsonString(found.Scheme), jsonString(pkg))
	for i, k := range keys {
		d := defs[k]
		if i > 0 {
			b.WriteString(",")
		}
		fmt.Fprintf(&b, "\n  %s: [\n   %s,\n   %d,\n   %d\n  ]", jsonString(k), jsonString(d.File), d.StartLine, d.EndLine)
	}
	if len(keys) > 0 {
		b.WriteString("\n ")
	}
	b.WriteString("}\n}\n")
	return b.String()
}

// jsonString encodes as JavaScript's JSON.stringify does: no HTML escaping.
func jsonString(s string) string {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.Encode(s)
	return strings.TrimSuffix(b.String(), "\n")
}

func deserialize(text string) (string, map[string]Definition, error) {
	var s stored
	if err := json.Unmarshal([]byte(text), &s); err != nil {
		return "", nil, err
	}
	defs := map[string]Definition{}
	for k, v := range s.Definitions {
		file, _ := v[0].(string)
		start, _ := v[1].(float64)
		end, _ := v[2].(float64)
		defs[k] = Definition{Symbol: k, File: file, StartLine: int(start), EndLine: int(end)}
	}
	scheme := s.Scheme
	if scheme == "" {
		scheme = "scip-typescript" // files written before seersucker
	}
	return scheme, defs, nil
}

// Found is what indexing one scheme's packages at one commit found:
// definitions by package. A package the indexer couldn't find is absent.
type Found struct {
	Indexer, Version, Scheme string
	Packages                 map[string]map[string]Definition
}

type Indexer func(repo *Repo, sha, scheme string, packages []string) (Found, error)

// DefaultIndexer runs the scheme's indexer on the commit, checked out on its own.
func DefaultIndexer(repo *Repo, sha, scheme string, packages []string) (Found, error) {
	info := IndexerFor(scheme)
	if info == nil {
		return Found{}, fmt.Errorf("no indexer for %s here (phoropter-index-%s isn't on PATH)", scheme, scheme)
	}
	tree, err := repo.Checkout(sha)
	if err != nil {
		return Found{}, err
	}
	defer RemoveDir(tree)
	out, err := RunIndexer(info.Command, info.Cwd, tree, packages)
	if err != nil {
		return Found{}, err
	}
	found := Found{Indexer: out.Indexer, Version: out.Version, Scheme: scheme, Packages: map[string]map[string]Definition{}}
	for _, pkg := range packages {
		if defs, ok := DefinitionsOf(out, pkg); ok {
			found.Packages[pkg] = defs
		}
	}
	return found, nil
}

// Lookup is what resolving at one commit gets: the definitions found, and the
// package keys that couldn't be indexed, whose anchors show no-index.
type Lookup struct {
	Definitions map[string]Definition
	Failed      map[string]string
}

type memo struct {
	defs map[string]Definition
	err  string
}

type Store struct {
	Repo    *Repo
	cache   string
	indexer Indexer
	memory  map[string]memo
	// Asked is every (sha, package key) a build asked for, so `phoropter index`
	// knows what to write. Keys in order of asking.
	Asked     map[string]map[string]bool
	AskedSHAs []string
}

func NewStore(repo *Repo, cache string, indexer Indexer) *Store {
	if cache == "" {
		cache = CacheDir()
	}
	if indexer == nil {
		indexer = DefaultIndexer
	}
	// One cache per repository, keyed by its path.
	sum := sha256.Sum256([]byte(repo.Dir))
	return &Store{Repo: repo, cache: filepath.Join(cache, "index", hex.EncodeToString(sum[:])[:16]), indexer: indexer, memory: map[string]memo{}, Asked: map[string]map[string]bool{}}
}

func (s *Store) CommittedPath(sha, key string) string {
	_, pkg := splitKey(key)
	return filepath.Join(s.Repo.Dir, StoreDir, sha, fileOf(pkg))
}

func (s *Store) cachedPath(sha, key string) string {
	_, pkg := splitKey(key)
	return filepath.Join(s.cache, sha, fileOf(pkg))
}

func (s *Store) IsCommitted(sha, key string) bool {
	_, ok := s.readAt(s.CommittedPath(sha, key), key)
	return ok
}

// Lookup gives the definitions at one commit for these package keys: from the
// committed store, then the cache, then each scheme's indexer, run once for
// whatever of its packages is missing, filling the cache.
func (s *Store) Lookup(sha string, keys map[string]bool) *Lookup {
	l := &Lookup{Definitions: map[string]Definition{}, Failed: map[string]string{}}
	missing := map[string][]string{}
	var schemes []string
	sorted := make([]string, 0, len(keys))
	for k := range keys {
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)
	for _, key := range sorted {
		if s.Asked[sha] == nil {
			s.Asked[sha] = map[string]bool{}
			s.AskedSHAs = append(s.AskedSHAs, sha)
		}
		s.Asked[sha][key] = true
		m, ok := s.memory[sha+" "+key]
		if !ok {
			if defs, found := s.read(sha, key); found {
				m, ok = memo{defs: defs}, true
			}
		}
		switch {
		case ok && m.err != "":
			l.Failed[key] = m.err
		case ok:
			for k, d := range m.defs {
				l.Definitions[k] = d
			}
		default:
			scheme, pkg := splitKey(key)
			if missing[scheme] == nil {
				schemes = append(schemes, scheme)
			}
			missing[scheme] = append(missing[scheme], pkg)
		}
	}
	for _, scheme := range schemes {
		pkgs := missing[scheme]
		found, err := s.indexer(s.Repo, sha, scheme, pkgs)
		for _, pkg := range pkgs {
			key := scheme + " " + pkg
			var m memo
			switch defs, ok := found.Packages[pkg]; {
			case err != nil:
				m.err = strings.SplitN(err.Error(), "\n", 2)[0]
			case !ok:
				m.err = fmt.Sprintf("%s found no package %s", found.Indexer, pkg)
			default:
				m.defs = defs
				s.write(s.cachedPath(sha, key), Serialize(found, pkg, defs))
			}
			s.memory[sha+" "+key] = m
			if m.err != "" {
				l.Failed[key] = m.err
				continue
			}
			for k, d := range m.defs {
				l.Definitions[k] = d
			}
		}
	}
	return l
}

func (s *Store) readAt(path, key string) (map[string]Definition, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	scheme, defs, err := deserialize(string(b))
	want, _ := splitKey(key)
	if err != nil || scheme != want {
		return nil, false
	}
	return defs, true
}

func (s *Store) read(sha, key string) (map[string]Definition, bool) {
	for _, p := range []string{s.CommittedPath(sha, key), s.cachedPath(sha, key)} {
		if defs, ok := s.readAt(p, key); ok {
			s.memory[sha+" "+key] = memo{defs: defs}
			return defs, true
		}
	}
	return nil, false
}

// Commit is `phoropter index`: it commits the index of one (sha, package key)
// into the repository, from the cache or by indexing. It returns the path
// written, or "" when it was already there or couldn't be indexed.
func (s *Store) Commit(sha, key string) string {
	path := s.CommittedPath(sha, key)
	if s.IsCommitted(sha, key) {
		return ""
	}
	s.Lookup(sha, map[string]bool{key: true})
	b, err := os.ReadFile(s.cachedPath(sha, key))
	if err != nil {
		return ""
	}
	s.write(path, string(b))
	return path
}

func (s *Store) write(path, text string) {
	os.MkdirAll(filepath.Dir(path), 0o755)
	os.WriteFile(path, []byte(text), 0o644)
}
