package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
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
	exe := filepath.Join(t.TempDir(), "phoropter")
	if out, err := exec.Command("go", "build", "-o", exe, ".").CombinedOutput(); err != nil {
		t.Fatal(string(out))
	}
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
