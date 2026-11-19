package site

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The batch answers what git show and rev-parse did, including what isn't
// there, and a batch that dies mid-session costs a slow read, not a wrong
// one (jaconet).
func TestBatchReadsAsGitDoes(t *testing.T) {
	dir := t.TempDir()
	git := func(args ...string) string {
		out, err := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@t"}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatal(string(out))
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-q")
	os.MkdirAll(filepath.Join(dir, "d"), 0o755)
	os.WriteFile(filepath.Join(dir, "d", "a file.txt"), []byte("one\ntwo"), 0o644)
	git("add", ".")
	git("commit", "-q", "-m", "first")
	head := git("rev-parse", "HEAD")
	repo := &Repo{Dir: dir}

	check := func(when string) {
		t.Helper()
		if got := repo.Resolve("HEAD"); got != head {
			t.Errorf("%s: Resolve(HEAD) = %q", when, got)
		}
		if got := repo.Resolve("none, a new repo"); got != "" {
			t.Errorf("%s: prose resolved to %q", when, got)
		}
		if text, ok := repo.Show(head, "d/a file.txt"); !ok || text != "one\ntwo" {
			t.Errorf("%s: Show = %q, %v", when, text, ok)
		}
		if _, ok := repo.Show(head, "nope.txt"); ok {
			t.Errorf("%s: a missing file was found", when)
		}
		want, _ := exec.Command("git", "-C", dir, "show", head+":d").Output()
		if text, ok := repo.Show(head, "d"); !ok || text != string(want) {
			t.Errorf("%s: a directory shows as %q, git shows %q", when, text, want)
		}
	}
	check("first")
	if repo.batch == nil {
		t.Fatal("no batch was started")
	}
	repo.batch.cmd.Process.Kill()
	repo.batch.cmd.Wait()
	check("after the batch died")
	check("with a fresh batch")
}
