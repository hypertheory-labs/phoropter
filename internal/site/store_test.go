package site

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// batik: indexes are facts about commits, stored apart from pages. Against a
// clone of mock-control, so committing an index never touches the real one.

const pkgKey = "scip-typescript @hypertheory-labs/mock-control"

func clone(t *testing.T) *Repo {
	t.Helper()
	if _, err := os.Stat(mockControl); err != nil {
		t.Skip("no ../mock-control")
	}
	dir := t.TempDir()
	if out, err := exec.Command("git", "clone", "-q", mockControl, dir).CombinedOutput(); err != nil {
		t.Fatal(string(out))
	}
	return &Repo{Dir: dir}
}

func symbolViews(t *testing.T, repo *Repo, store *Store) []string {
	t.Helper()
	s, err := BuildSite(repo, "f02528f", store)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range s.Plans {
		if p.Name != "pumice" {
			continue
		}
		for _, e := range p.Doc.Entries {
			if e.Slug == "selectors-use-the-library-s-prefix-mock-panel" {
				var out []string
				for _, v := range e.Resolved[0].Views {
					out = append(out, v.Role+" "+v.Status)
				}
				return out
			}
		}
	}
	t.Fatal("no pumice selectors entry")
	return nil
}

func counting(calls *[]string) Indexer {
	return func(repo *Repo, sha, scheme string, packages []string) (Found, error) {
		*calls = append(*calls, sha[:7])
		return DefaultIndexer(repo, sha, scheme, packages)
	}
}

func TestSecondBuildReadsTheCache(t *testing.T) {
	repo := clone(t)
	cache := t.TempDir()
	var first, second []string
	views := symbolViews(t, repo, NewStore(repo, cache, counting(&first)))
	if len(first) == 0 {
		t.Fatal("the first build indexed nothing")
	}
	if got := symbolViews(t, repo, NewStore(repo, cache, counting(&second))); !reflect.DeepEqual(got, views) || len(second) != 0 {
		t.Errorf("second build: %v, indexed %v", got, second)
	}
}

func TestIndexCommitsWhatPlansNeed(t *testing.T) {
	repo := clone(t)
	store := NewStore(repo, t.TempDir(), nil)
	views := symbolViews(t, repo, store)
	base := repo.Resolve("da023f1")
	path := store.Commit(base, pkgKey)
	if path != filepath.Join(repo.Dir, ".phoropter/index", base, "@hypertheory-labs+mock-control.json") {
		t.Fatalf("path = %s", path)
	}
	if store.Commit(base, pkgKey) != "" {
		t.Error("already committed")
	}
	b, _ := os.ReadFile(path)
	text := string(b)
	if strings.Contains(text, base) || !strings.HasPrefix(text, "{\n \"indexer\": ") || !strings.Contains(text, "\"scheme\": \"scip-typescript\"") {
		t.Errorf("stored file:\n%.300s", text)
	}
	// Everything the anchors asked for committed, a fresh cache, and an
	// indexer that would fail: the build reads the store alone.
	for _, sha := range store.AskedSHAs {
		var keys []string
		for k := range store.Asked[sha] {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			store.Commit(sha, k)
		}
	}
	broken := func(*Repo, string, string, []string) (Found, error) {
		return Found{}, errors.New("scip-typescript is not installed")
	}
	if got := symbolViews(t, repo, NewStore(repo, t.TempDir(), broken)); !reflect.DeepEqual(got, views) {
		t.Errorf("from the store alone: %v, want %v", got, views)
	}
}

func TestFailingIndexerIsNoIndex(t *testing.T) {
	repo := clone(t)
	broken := func(*Repo, string, string, []string) (Found, error) {
		return Found{}, errors.New("scip-typescript is not installed")
	}
	if got := symbolViews(t, repo, NewStore(repo, t.TempDir(), broken)); !reflect.DeepEqual(got, []string{"base no-index", "as-built head no-index", "generated no-index"}) {
		t.Errorf("views = %v", got)
	}
}
