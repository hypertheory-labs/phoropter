package site

import (
	"archive/tar"
	"bytes"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
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

// go run adds "exit status 1" below what the program said; the indexer's own
// last line is the error (grosgrain).
func TestLastLineSkipsGoRunsExitStatus(t *testing.T) {
	for msg, want := range map[string]string{
		"scip-go isn't installed: go install x\nexit status 1\n": "scip-go isn't installed: go install x",
		"one\ntwo":        "two",
		"exit status 2":   "exit status 2",
		"  only line  \n": "only line",
	} {
		if got := lastLine(msg); got != want {
			t.Errorf("lastLine(%q) = %q, want %q", msg, got, want)
		}
	}
}

// Checkout unpacks in Go what `git archive | tar -x` did, so Windows needs no
// sh (chiffon): the same files, contents, executable bits and symlinks.
func TestCheckoutMatchesTar(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("no sh to compare against")
	}
	src := t.TempDir()
	git := func(args ...string) {
		if out, err := exec.Command("git", append([]string{"-C", src, "-c", "user.name=t", "-c", "user.email=t@t"}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "-q")
	os.MkdirAll(filepath.Join(src, "a", "b"), 0o755)
	os.WriteFile(filepath.Join(src, "a", "b", "c.go"), []byte("package b\n"), 0o644)
	os.WriteFile(filepath.Join(src, "run.sh"), []byte("#!/bin/sh\n"), 0o755)
	os.Symlink("a/b/c.go", filepath.Join(src, "link.go"))
	git("add", ".")
	git("commit", "-qm", "x")
	r := &Repo{Dir: src}
	got, err := r.Checkout("HEAD")
	if err != nil {
		t.Fatal(err)
	}
	defer RemoveDir(got)
	want := t.TempDir()
	if out, err := exec.Command("sh", "-c", `git -C "$1" archive HEAD | tar -x -C "$2"`, "sh", src, want).CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	describe := func(root string) map[string]string {
		m := map[string]string{}
		filepath.Walk(root, func(p string, fi os.FileInfo, err error) error {
			rel, _ := filepath.Rel(root, p)
			switch {
			case fi.Mode()&os.ModeSymlink != 0:
				l, _ := os.Readlink(p)
				m[rel] = "-> " + l
			case fi.IsDir():
				m[rel] = "dir"
			default:
				b, _ := os.ReadFile(p)
				m[rel] = fmt.Sprintf("%v %q", fi.Mode().Perm()&0o100 != 0, b)
			}
			return nil
		})
		return m
	}
	g, w := describe(got), describe(want)
	if fmt.Sprint(g) != fmt.Sprint(w) {
		t.Errorf("Checkout:\n%v\ntar:\n%v", g, w)
	}
}

// An entry outside the checkout is refused, not written (chiffon).
func TestUntarRefusesEscapes(t *testing.T) {
	var b bytes.Buffer
	tw := tar.NewWriter(&b)
	tw.WriteHeader(&tar.Header{Name: "../out.txt", Mode: 0o644, Size: 1, Typeflag: tar.TypeReg})
	tw.Write([]byte("x"))
	tw.Close()
	dir := t.TempDir()
	if err := untar(&b, filepath.Join(dir, "in")); err == nil {
		t.Error("untar wrote ../out.txt")
	}
	if _, err := os.Stat(filepath.Join(dir, "out.txt")); err == nil {
		t.Error("out.txt exists outside the checkout")
	}
}

// Windows's "address in use" counts, as on any other system (chiffon).
func TestAddrInUseOnEverySystem(t *testing.T) {
	for _, e := range []error{syscall.EADDRINUSE, wsaeaddrinuse} {
		if !addrInUse(&net.OpError{Op: "listen", Err: os.NewSyscallError("bind", e)}) {
			t.Errorf("%v isn't address-in-use", e)
		}
	}
	if addrInUse(&net.OpError{Op: "listen", Err: os.NewSyscallError("bind", syscall.EACCES)}) {
		t.Error("a refused permission counted as address-in-use")
	}
}
