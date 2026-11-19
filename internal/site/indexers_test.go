package site

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// seersucker and dimity: indexers are programs, chosen by the scheme a
// symbol starts with, checked against indexers/README.md, and given a time
// limit.

var here = &Repo{Dir: "../.."}

func TestSymbolGrammar(t *testing.T) {
	for s, want := range map[string]bool{
		"scip-typescript npm @hypertheory-labs/mock-control . src/lib/panel/`mock-panel.ts`/MockPanel#": true,
		"scip-go gomod phoropter-tui . `phoropter-tui/internal/tui`/docScreen#annotate().":              true,
		"roslyn nuget Contoso.Orders . Contoso/Orders/OrderService#Place().":                            true,
		"README.md › Setup › 1. Install":                                                                false,
		"src/render.ts":                                                                                 false,
	} {
		if IsScipSymbol(s) != want {
			t.Errorf("IsScipSymbol(%q) != %v", s, want)
		}
	}
	if PackageKey("scip-go gomod phoropter-tui . `phoropter-tui/internal/tui`/form#") != "scip-go phoropter-tui" {
		t.Error("package key")
	}
	for s, want := range map[string]string{
		"scip-go gomod phoropter-tui . `phoropter-tui/internal/tui`/docScreen#annotate().": "annotate",
		"roslyn nuget Contoso.Orders . Contoso/Orders/OrderService#`.ctor`().":             ".ctor",
		"scip-typescript npm p . src/`mock-panel.ts`/MockPanel#":                           "MockPanel",
	} {
		if got := LastName(s); got != want {
			t.Errorf("LastName(%q) = %q, want %q", s, got, want)
		}
	}
}

func TestFirstPartyIndexersAreFound(t *testing.T) {
	if i := IndexerFor("scip-typescript"); i == nil || i.Indexer != "phoropter-index-typescript" {
		t.Errorf("typescript = %+v", i)
	}
	if IndexerFor("nobody-has-this") != nil {
		t.Error("found an indexer that doesn't exist")
	}
}

func fake(defs string) []string {
	return []string{"node", "-e", `
const out = { indexer: 'fake', version: '0', scheme: 'roslyn', extensions: ['.cs'] };
if (process.argv[1] === '--describe') { console.log(JSON.stringify(out)); process.exit(0) }
console.log(JSON.stringify({ ...out, packages: { p: { definitions: ` + defs + ` } } }))`, "--"}
}

func TestCheckSaysWhatsWrong(t *testing.T) {
	ok, report := CheckIndexer(fake(`{ 'roslyn npm p 1.0 src/Thing#': ['src/nowhere.ts', 1, 1], 'roslyn npm q . src/Other#': ['go.mod', 1, 1] }`), "", here)
	text := strings.Join(report, "\n")
	if ok || !strings.Contains(text, `version should be written as "."`) || !strings.Contains(text, "package q, but it's listed under p") {
		t.Errorf("report:\n%s", text)
	}
}

// A Windows path is named for what it is, not "isn't a file" (cambric).
func TestCheckNamesABackslashPath(t *testing.T) {
	ok, report := CheckIndexer(fake(`{ 'roslyn npm p . Site#': ['internal\\site\\site.go', 1, 1] }`), "", here)
	text := strings.Join(report, "\n")
	if ok || !strings.Contains(text, `internal\site\site.go uses \; the contract's paths use /`) {
		t.Errorf("report:\n%s", text)
	}
}

// Names an indexer makes up (.ctor) aren't looked for, and a few names a
// language spells differently (op_Addition) are notes. Wrong ranges still
// fail, because then nearly every name is missing.
func TestCheckReadsMadeUpNamesAndCatchesBadRanges(t *testing.T) {
	text, _ := here.Show(here.Resolve("HEAD"), "internal/site/site.go")
	lines := strings.Split(text, "\n")
	at := func(needle string) int {
		for i, l := range lines {
			if strings.Contains(l, needle) {
				return i + 1
			}
		}
		return 0
	}
	names := []string{"BuildSite", "PackagesOf", "ResolveEntries", "commitOf", "placeKey", "placeFor", "addedTime", "hasSHA", "localeLess", "IndexFor", "Commit", "Site", "Context", "Plan", "Doc", "Place", "PlaceRef", "ResolvedEntry", "EntryRef", "ResolvedAnchor"}
	var good, wrong []string
	for _, n := range names {
		good = append(good, fmt.Sprintf(`'roslyn nuget p . Thing#%s().': ['internal/site/site.go', 1, %d]`, n, len(lines)))
		wrong = append(wrong, fmt.Sprintf(`'roslyn nuget p . Thing#%s().': ['internal/site/site.go', 1, 1]`, n))
	}
	fn := at("func BuildSite(")
	good = append(good, fmt.Sprintf("'roslyn nuget p . Store#`.ctor`().': ['internal/site/site.go', %d, %d]", fn, fn+5), fmt.Sprintf(`'roslyn nuget p . Store#op_Addition().': ['internal/site/site.go', %d, %d]`, fn, fn))
	ok, report := CheckIndexer(fake("{"+strings.Join(good, ",")+"}"), "", here)
	if !ok || !strings.Contains(strings.Join(report, "\n"), "1 of 21 name(s) aren't spelled in their ranges") {
		t.Errorf("good:\n%s", strings.Join(report, "\n"))
	}
	if ok, _ := CheckIndexer(fake("{"+strings.Join(wrong, ",")+"}"), "", here); ok {
		t.Error("wrong ranges passed")
	}
}

func TestIndexerTimesOut(t *testing.T) {
	t.Setenv("PHOROPTER_INDEX_TIMEOUT", "1")
	_, err := RunIndexer([]string{"node", "-e", "setTimeout(() => {}, 60000)", "--"}, "", ".", nil)
	if err == nil || !strings.Contains(err.Error(), "timed out after 1s") {
		t.Errorf("err = %v", err)
	}
}

func TestBothFirstPartyIndexersPassCheck(t *testing.T) {
	if _, err := exec.LookPath("scip-go"); err != nil {
		if _, err := exec.LookPath(strings.TrimSpace(mustOut("go", "env", "GOPATH")) + "/bin/scip-go"); err != nil {
			t.Skip("scip-go not installed")
		}
	}
	for _, info := range Indexers() {
		if info.Cwd == "" {
			continue // on PATH, not first-party
		}
		repo := here
		if info.Scheme == "scip-typescript" {
			// phoropter has had no TypeScript project since organdy, so this
			// one gets a small repository of its own. The check reads git, which
			// go test's cache can't see: a cached pass hid this (poplin).
			repo = tsRepo(t)
		}
		ok, report := CheckIndexer(info.Command, info.Cwd, repo)
		if !ok {
			t.Errorf("%s:\n%s", info.Indexer, strings.Join(report, "\n"))
		}
	}
}

func tsRepo(t *testing.T) *Repo {
	dir := t.TempDir()
	files := map[string]string{
		"package.json":  `{ "name": "ts-sample", "version": "1.0.0" }`,
		"tsconfig.json": `{ "compilerOptions": { "strict": true }, "include": ["src"] }`,
		"src/panel.ts":  "export class Panel {\n  open(): void {\n    console.log('open')\n  }\n}\n\nexport function make(): Panel {\n  return new Panel()\n}\n",
	}
	for name, body := range files {
		os.MkdirAll(filepath.Join(dir, filepath.Dir(name)), 0o755)
		os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644)
	}
	for _, args := range [][]string{{"init", "-q"}, {"add", "."}, {"-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "-m", "sample"}} {
		if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatal(string(out))
		}
	}
	return &Repo{Dir: dir}
}

func mustOut(name string, args ...string) string {
	b, _ := exec.Command(name, args...).Output()
	return string(b)
}
