// Package site is phoropter (organdy): it reads one repository at one commit
// through git, and builds the site of its plans, their anchors and the places
// they point at. The HTML it writes is the API.
package site

import (
	"bytes"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// A Repo is read through git at a commit, never the working tree, so the site
// is a function of the commit alone.
type Repo struct {
	Dir string
}

func (r *Repo) git(args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", r.Dir}, args...)...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	return out.String(), err
}

// Resolve gives the full sha, or "" when ref isn't a commit here (a plan's
// base can be prose, like gimlet's "none, a new repo").
func (r *Repo) Resolve(ref string) string {
	out, err := r.git("rev-parse", "--verify", "--quiet", ref+"^{commit}")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// Short is the shortest unambiguous form, at least 7 characters, as git shows it.
func (r *Repo) Short(sha string) string {
	out, _ := r.git("rev-parse", "--short", sha)
	return strings.TrimSpace(out)
}

func (r *Repo) Files(sha, dir string) []string {
	args := []string{"ls-tree", "-r", "--name-only", sha}
	if dir != "" {
		args = append(args, "--", dir)
	}
	out, _ := r.git(args...)
	var files []string
	for _, f := range strings.Split(out, "\n") {
		if f != "" {
			files = append(files, f)
		}
	}
	return files
}

// Added is the commit that first added path, as seen from sha, and its time.
func (r *Repo) Added(sha, path string) (string, int64, bool) {
	out, _ := r.git("log", "--diff-filter=A", "--format=%H %ct", "--reverse", sha, "--", path)
	first := strings.SplitN(out, "\n", 2)[0]
	if first == "" {
		return "", 0, false
	}
	parts := strings.SplitN(first, " ", 2)
	t, _ := strconv.ParseInt(parts[1], 10, 64)
	return parts[0], t, true
}

// Show is a file's text at a commit, and whether it exists there.
func (r *Repo) Show(sha, path string) (string, bool) {
	out, err := r.git("show", sha+":"+path)
	if err != nil {
		return "", false
	}
	return out, true
}

// Checkout writes the tree at sha into a fresh temp directory, for an
// indexer. The caller removes it with RemoveDir.
func (r *Repo) Checkout(sha string) (string, error) {
	dir, err := os.MkdirTemp("", "phoropter-")
	if err != nil {
		return "", err
	}
	cmd := exec.Command("sh", "-c", `git -C "$1" archive "$2" | tar -x -C "$3"`, "sh", r.Dir, sha, dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		os.RemoveAll(dir)
		return "", &exec.ExitError{Stderr: out}
	}
	return dir, nil
}

func RemoveDir(dir string) { os.RemoveAll(dir) }
