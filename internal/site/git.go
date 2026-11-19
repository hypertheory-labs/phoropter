// Package site is phoropter (organdy): it reads one repository at one commit
// through git, and builds the site of its plans, their anchors and the places
// they point at. The HTML it writes is the API.
package site

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

// A Repo is read through git at a commit, never the working tree, so the site
// is a function of the commit alone.
type Repo struct {
	Dir string

	// One git cat-file --batch answers Show and Resolve, which were 720 of a
	// build's 750 git processes (jaconet). Started on first use, kept for the
	// Repo's life, behind mu.
	mu    sync.Mutex
	batch *catFile
}

type catFile struct {
	cmd *exec.Cmd
	in  io.WriteCloser
	out *bufio.Reader
}

// object asks the batch for one object by name: its sha, type and contents, or
// ok false when git says it's missing. A read that can't go through the
// batch says so with used false, and the caller runs the command it
// replaced; the next read starts a fresh batch.
func (r *Repo) object(name string) (sha, kind string, content []byte, ok, used bool) {
	if strings.ContainsAny(name, "\n\r") {
		return "", "", nil, false, false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.batch == nil {
		cmd := exec.Command("git", "-C", r.Dir, "cat-file", "--batch")
		in, err1 := cmd.StdinPipe()
		out, err2 := cmd.StdoutPipe()
		if err1 != nil || err2 != nil || cmd.Start() != nil {
			return "", "", nil, false, false
		}
		r.batch = &catFile{cmd: cmd, in: in, out: bufio.NewReader(out)}
	}
	b := r.batch
	fail := func() (string, string, []byte, bool, bool) {
		b.in.Close()
		b.cmd.Wait()
		r.batch = nil
		return "", "", nil, false, false
	}
	if _, err := io.WriteString(b.in, name+"\n"); err != nil {
		return fail()
	}
	header, err := b.out.ReadString('\n')
	if err != nil {
		return fail()
	}
	// "<sha> <type> <size>" for an object; "<name> missing" (or ambiguous)
	// otherwise, and the name can contain spaces.
	fields := strings.Fields(header)
	size := -1
	if len(fields) == 3 {
		size, _ = strconv.Atoi(fields[2])
	}
	if size < 0 || strings.HasSuffix(header, " missing\n") || strings.HasSuffix(header, " ambiguous\n") {
		return "", "", nil, false, true
	}
	content = make([]byte, size+1) // the object, then a newline
	if _, err := io.ReadFull(b.out, content); err != nil {
		return fail()
	}
	return fields[0], fields[1], content[:size], true, true
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
	if sha, kind, _, ok, used := r.object(ref + "^{commit}"); used {
		if !ok || kind != "commit" {
			return ""
		}
		return sha
	}
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
	if _, kind, content, ok, used := r.object(sha + ":" + path); used {
		if !ok {
			return "", false
		}
		if kind == "blob" {
			return string(content), true
		}
		// A tree shows as a listing, not its raw object: leave it to git show.
	}
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
