package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hypertheory-labs/phoropter/internal/site"
)

func TestRepoRootWalksUpToGit(t *testing.T) {
	root := t.TempDir()
	os.Mkdir(filepath.Join(root, ".git"), 0o755)
	os.MkdirAll(filepath.Join(root, "a", "b"), 0o755)
	if got := repoRoot(filepath.Join(root, "a", "b")); got != root {
		t.Errorf("root = %s", got)
	}
	if repoRoot(t.TempDir()) != "" {
		t.Error("found a repository where there's none")
	}
}

// One file per server: a second server for the same repository that has since
// stopped is tidied away, and the live one's record is untouched.
func TestServingURLFindsTheLiveServer(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	root := t.TempDir()
	if servingURL(root) != "" {
		t.Fatal("nothing is serving")
	}
	write := func(pid int, url string, started int64) string {
		f := servingFile(root, pid)
		os.MkdirAll(filepath.Dir(f), 0o755)
		b, _ := json.Marshal(map[string]any{"url": url, "root": root, "pid": pid, "started": started})
		os.WriteFile(f, b, 0o644)
		return f
	}
	mine := write(os.Getpid(), "http://localhost:4174/", 1)
	gone := write(1<<22+7, "http://localhost:4176/", 2)
	if got := servingURL(root); got != "http://localhost:4174/" {
		t.Errorf("url = %s", got)
	}
	if _, err := os.Stat(gone); err == nil {
		t.Error("the stopped server's file wasn't tidied")
	}
	if _, err := os.Stat(mine); err != nil {
		t.Error("the live server's file is gone")
	}
}

// dimity: two indexers for one extension list their symbols under a heading
// each, and --scheme picks one. Fakes on a PATH of their own.
func TestSymbolsGroupsBySchemeAndPicksOne(t *testing.T) {
	bin := t.TempDir()
	for _, scheme := range []string{"alpha", "beta"} {
		script := "#!/usr/bin/env node\nconst out = { indexer: '" + scheme + "-indexer', version: '0', scheme: '" + scheme + "', extensions: ['.go'] };\nif (process.argv[2] === '--describe') { console.log(JSON.stringify(out)); process.exit(0) }\nconsole.log(JSON.stringify({ ...out, packages: { p: { definitions: { '" + scheme + " gomod p . main#': ['main.go', 1, 2] } } } }));\n"
		os.WriteFile(filepath.Join(bin, "phoropter-index-"+scheme), []byte(script), 0o755)
	}
	exe := build(t)
	run := func(args ...string) string {
		cmd := exec.Command(exe, append([]string{"symbols", "main.go"}, args...)...)
		cmd.Env = append(os.Environ(), "PATH="+bin+":"+os.Getenv("PATH"), "PHOROPTER_INDEXERS="+t.TempDir())
		out, _ := cmd.CombinedOutput()
		return string(out)
	}
	both := run()
	if !strings.Contains(both, "alpha (alpha-indexer):\n  1–2        alpha gomod p . main#") || !strings.Contains(both, "beta (beta-indexer):\n") {
		t.Errorf("both:\n%s", both)
	}
	if one := run("--scheme", "beta"); strings.Contains(one, "alpha") || !strings.HasPrefix(one, "1–2") {
		t.Errorf("one:\n%s", one)
	}
}

func build(t *testing.T) string {
	exe := filepath.Join(t.TempDir(), "phoropter")
	if out, err := exec.Command("go", "build", "-o", exe, ".").CombinedOutput(); err != nil {
		t.Fatal(string(out))
	}
	return exe
}

// poplin: "nothing defined" says what phoropter can tell. The course
// repository's shape: two copies of one project, diverged, and only the first
// path's indexed; a Program.cs of top-level statements; and a file with
// changes that aren't committed.
func TestSymbolsSaysWhyNothingIsDefined(t *testing.T) {
	repo := t.TempDir()
	files := map[string]string{
		"instructor/Software.Api/Vendors/Api.cs": "class Api {\n  void Get() {}\n  void Post() {}\n}\n",
		"student/Software.Api/Vendors/Api.cs":    "class Api {\n  void Get() {}\n}\n",
		"instructor/Software.Api/Program.cs":     "var app = Build();\napp.Run();\n",
	}
	for name, body := range files {
		os.MkdirAll(filepath.Join(repo, filepath.Dir(name)), 0o755)
		os.WriteFile(filepath.Join(repo, name), []byte(body), 0o644)
	}
	for _, args := range [][]string{{"init", "-q"}, {"add", "."}, {"-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "-m", "copies"}} {
		if out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).CombinedOutput(); err != nil {
			t.Fatal(string(out))
		}
	}
	bin := t.TempDir()
	script := `#!/bin/sh
if [ "$1" = --describe ]; then echo '{"indexer":"fake","version":"0","scheme":"fake","extensions":[".cs"]}'; exit 0; fi
echo '{"indexer":"fake","version":"0","scheme":"fake","packages":{"Software.Api":{"definitions":{"fake nuget Software.Api . Software/Api/Vendors/Api#":["instructor/Software.Api/Vendors/Api.cs",1,4]}}}}'
`
	os.WriteFile(filepath.Join(bin, "phoropter-index-fake"), []byte(script), 0o755)
	exe := build(t)
	run := func(file string) string {
		cmd := exec.Command(exe, "symbols", filepath.Join(repo, file), "--scheme", "fake")
		cmd.Env = append(os.Environ(), "PATH="+bin+":"+os.Getenv("PATH"), "PHOROPTER_INDEXERS="+t.TempDir())
		out, _ := cmd.CombinedOutput()
		return string(out)
	}
	if out := run("student/Software.Api/Vendors/Api.cs"); !strings.Contains(out, "instructor/Software.Api/Vendors/Api.cs is indexed instead") {
		t.Errorf("the losing copy:\n%s", out)
	}
	if out := run("instructor/Software.Api/Program.cs"); !strings.Contains(out, "no named declarations") || strings.Contains(out, "uncommitted") {
		t.Errorf("top-level statements:\n%s", out)
	}
	os.WriteFile(filepath.Join(repo, "student/Software.Api/Vendors/Api.cs"), []byte("class Api {}\n"), 0o644)
	if out := run("student/Software.Api/Vendors/Api.cs"); !strings.Contains(out, "changes that aren't committed") {
		t.Errorf("a changed file:\n%s", out)
	}
}

func TestCopyOf(t *testing.T) {
	indexed := []string{"a/x/Vendors/Api.cs", "b/Other/Api.cs", "Program.cs"}
	for rel, want := range map[string]string{
		"s/x/Vendors/Api.cs": "a/x/Vendors/Api.cs",
		"s/y/Vendors/Api.cs": "a/x/Vendors/Api.cs",
		"s/Api.cs":           "",
		"s/Program.cs":       "",
	} {
		if got := copyOf(rel, indexed); got != want {
			t.Errorf("copyOf(%s) = %q, want %q", rel, got, want)
		}
	}
}

// init writes the template as the one draft, based at the commit given, and
// never replaces a draft that's there (muslin).
func TestInitWritesTheFirstDraftOnce(t *testing.T) {
	root := t.TempDir()
	if err := writeFirstDraft(root, "abc1234"); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(root, draftPath))
	draft := string(b)
	if !strings.HasPrefix(draft, "---\nbase: abc1234\n---\n") || strings.Contains(draft, "{{") {
		t.Errorf("base not filled in:\n%.80s", draft)
	}
	if !strings.Contains(draft, "# How we work here") {
		t.Error("not the template")
	}
	os.WriteFile(filepath.Join(root, draftPath), []byte("someone's draft"), 0o644)
	if err := writeFirstDraft(root, "abc1234"); err != errDraftExists {
		t.Errorf("err = %v, want errDraftExists", err)
	}
	if b, _ := os.ReadFile(filepath.Join(root, draftPath)); string(b) != "someone's draft" {
		t.Error("replaced an existing draft")
	}
}

// The walkthrough names the instructions file the repository has, carries
// the pointer, and gives the address when a server is running.
func TestInitWalkthrough(t *testing.T) {
	root := t.TempDir()
	if got := instructionsFile(root); !strings.Contains(got, "CLAUDE.md or AGENTS.md") {
		t.Errorf("no instructions: %q", got)
	}
	os.WriteFile(filepath.Join(root, "AGENTS.md"), nil, 0o644)
	if got := instructionsFile(root); got != "AGENTS.md" {
		t.Errorf("AGENTS.md only: %q", got)
	}
	os.MkdirAll(filepath.Join(root, "plans"), 0o755)
	for _, f := range []string{"2026-01-01-a.md", "2026-01-01-a.as-built.md", "2026-01-02-b.md"} {
		os.WriteFile(filepath.Join(root, "plans", f), nil, 0o644)
	}
	served := initWalkthrough(root, "http://localhost:4191/", countPlans(root))
	for _, want := range []string{"Paste this into AGENTS.md", "## Plans (phoropter)", "http://localhost:4191/draft/", "already has 2 plans"} {
		if !strings.Contains(served, want) {
			t.Errorf("walkthrough lacks %q:\n%s", want, served)
		}
	}
	if fresh := initWalkthrough(t.TempDir(), "", 0); strings.Contains(fresh, "already has") || !strings.Contains(fresh, "Run phoropter") {
		t.Errorf("fresh repository:\n%s", fresh)
	}
}

// The README shows the pointer init prints, and the two mustn't drift apart:
// a second copy that disagrees is what muslin set out to end.
func TestREADMEShowsThePointerInitPrints(t *testing.T) {
	b, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), pointer) {
		t.Error("README.md's pointer isn't the one phoropter init prints")
	}
}

// doctor names the fix at the level a newcomer is stuck at: no clone, a
// clone without the toolchain, or the toolchain without the tool (grosgrain).
func TestDoctorSaysTheFixForEachLevel(t *testing.T) {
	needsGo := map[string]int{"scip-go": 2}
	cases := []struct {
		name  string
		facts doctorFacts
		want  string
	}{
		{"no clone, Go", doctorFacts{git: "git", schemes: needsGo}, "fix: go install github.com/hypertheory-labs/phoropter/cmd/phoropter-index-scip-go@latest"},
		{"no clone, TypeScript", doctorFacts{git: "git", schemes: map[string]int{"scip-typescript": 1}}, "git clone https://github.com/hypertheory-labs/phoropter"},
		{"no Go", doctorFacts{git: "git", firstParty: "/p/indexers", schemes: needsGo,
			unavailable: []site.Unavailable{{Where: "/p/indexers/go"}}}, "runs on Go 1.25"},
		{"no scip-go", doctorFacts{git: "git", firstParty: "/p/indexers", schemes: needsGo,
			indexers: []site.IndexerInfo{{Indexer: "phoropter-index-go", Scheme: "scip-go",
				Needs: []site.Need{{What: "scip-go", Install: "go install scip-go"}}}}}, "fix: go install scip-go"},
		{"someone else's scheme", doctorFacts{git: "git", firstParty: "/p/indexers", schemes: map[string]int{"roslyn": 1}},
			"phoropter-index-roslyn on PATH"},
	}
	for _, c := range cases {
		out, missing := c.facts.report()
		if !missing || !strings.Contains(out, c.want) {
			t.Errorf("%s: missing=%v, want %q in:\n%s", c.name, missing, c.want, out)
		}
	}
	installed := doctorFacts{git: "git", schemes: needsGo,
		indexers: []site.IndexerInfo{{Indexer: "phoropter-index-go", Scheme: "scip-go"}}}
	if out, missing := installed.report(); missing || strings.Contains(out, "isn't in a clone") {
		t.Errorf("installed without a clone: missing=%v, and no clone is needed\n%s", missing, out)
	}
	ready := doctorFacts{git: "git", firstParty: "/p/indexers", schemes: needsGo,
		indexers: []site.IndexerInfo{{Indexer: "phoropter-index-go", Scheme: "scip-go"}}}
	if out, missing := ready.report(); missing || !strings.Contains(out, "ok    scip-go, 2 symbol anchors") {
		t.Errorf("ready: missing=%v\n%s", missing, out)
	}
	if out, missing := (doctorFacts{git: "git", schemes: map[string]int{}}).report(); missing || !strings.Contains(out, "no indexer is needed yet") {
		t.Errorf("no symbols: missing=%v\n%s", missing, out)
	}
}
