package site

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Finding and running indexers (seersucker). An indexer is any program that,
// given a checked-out repository and package names, prints definitions as
// JSON; indexers/README.md is the contract. The scheme an anchor's symbol
// starts with picks the indexer:
//
//  1. a first-party one in indexers/<name>/ of phoropter's own repository, when
//     phoropter runs from it, whose indexer.json gives the command that runs it;
//  2. otherwise a program on PATH named phoropter-index-<scheme>, the way gh
//     and kubectl find extensions. Nothing to configure.

type IndexerInfo struct {
	Indexer    string   `json:"indexer"`
	Version    string   `json:"version"`
	Scheme     string   `json:"scheme"`
	Extensions []string `json:"extensions"`
	Command    []string `json:"-"`
	Cwd        string   `json:"-"`
}

type IndexerOutput struct {
	Indexer  string `json:"indexer"`
	Version  string `json:"version"`
	Scheme   string `json:"scheme"`
	Packages map[string]struct {
		Definitions map[string][3]interface{} `json:"definitions"`
	} `json:"packages"`
	Missing []string `json:"missing"`
}

// FirstParty is where first-party indexers are: indexers/ in the repository
// phoropter was built from, found from the running binary (and set by tests).
var FirstParty = firstPartyDir()

func firstPartyDir() string {
	if dir := os.Getenv("PHOROPTER_INDEXERS"); dir != "" {
		return dir
	}
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	for dir := filepath.Dir(exe); ; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "indexers", "README.md")); err == nil {
			return filepath.Join(dir, "indexers")
		}
		if filepath.Dir(dir) == dir {
			return ""
		}
	}
}

// Describe is what a program says about itself with --describe.
func Describe(command []string, cwd string) (IndexerInfo, error) {
	cmd := exec.Command(command[0], append(command[1:], "--describe")...)
	cmd.Dir = cwd
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return IndexerInfo{}, fmt.Errorf("%s --describe failed: %s", strings.Join(command, " "), firstLine(errb.String()))
		}
		return IndexerInfo{}, fmt.Errorf("couldn't run %s: %v", strings.Join(command, " "), err)
	}
	var info IndexerInfo
	if err := json.Unmarshal(out.Bytes(), &info); err != nil {
		return IndexerInfo{}, fmt.Errorf("%s --describe printed something that isn't JSON", strings.Join(command, " "))
	}
	return info, nil
}

var (
	known     []IndexerInfo
	knownOnce sync.Once
)

// Indexers is every indexer this machine has: first-party ones, then any on PATH.
func Indexers() []IndexerInfo {
	knownOnce.Do(func() {
		if FirstParty != "" {
			entries, _ := os.ReadDir(FirstParty)
			for _, e := range entries {
				dir := filepath.Join(FirstParty, e.Name())
				b, err := os.ReadFile(filepath.Join(dir, "indexer.json"))
				if err != nil {
					continue
				}
				var m struct {
					Command []string `json:"command"`
				}
				if json.Unmarshal(b, &m) != nil || len(m.Command) == 0 {
					continue
				}
				if info, err := Describe(m.Command, dir); err == nil {
					info.Command, info.Cwd = m.Command, dir
					known = append(known, info)
				}
				// otherwise its toolchain isn't here; its anchors show no-index
			}
		}
		for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
			entries, err := os.ReadDir(dir)
			if err != nil {
				continue
			}
			for _, e := range entries {
				name := e.Name()
				if !strings.HasPrefix(name, "phoropter-index-") {
					continue
				}
				scheme := strings.TrimPrefix(name, "phoropter-index-")
				if hasScheme(known, scheme) {
					continue
				}
				path := filepath.Join(dir, name)
				if st, err := os.Stat(path); err != nil || st.IsDir() {
					continue
				}
				if info, err := Describe([]string{path}, ""); err == nil {
					info.Command = []string{path}
					known = append(known, info)
				}
			}
		}
	})
	return known
}

func hasScheme(infos []IndexerInfo, scheme string) bool {
	for _, i := range infos {
		if i.Scheme == scheme {
			return true
		}
	}
	return false
}

func IndexerFor(scheme string) *IndexerInfo {
	for _, i := range Indexers() {
		if i.Scheme == scheme {
			i := i
			return &i
		}
	}
	return nil
}

// IndexTimeout is how long an indexer may run before phoropter stops it
// (dimity): 5 minutes, or PHOROPTER_INDEX_TIMEOUT seconds. Anything longer is
// more likely stuck, like a package restore retrying with no network, than slow.
func IndexTimeout() time.Duration {
	if s, err := strconv.ParseFloat(os.Getenv("PHOROPTER_INDEX_TIMEOUT"), 64); err == nil && s > 0 {
		return time.Duration(s * float64(time.Second))
	}
	return 5 * time.Minute
}

// RunIndexer runs an indexer on a checked-out directory, for some packages
// (all of them when none are named). It fails with the indexer's own last
// line, or says it timed out.
func RunIndexer(command []string, cwd, dir string, packages []string) (IndexerOutput, error) {
	timeout := IndexTimeout()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, command[0], append(append(append([]string{}, command[1:]...), dir), packages...)...)
	cmd.Dir = cwd
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return IndexerOutput{}, fmt.Errorf("timed out after %gs (PHOROPTER_INDEX_TIMEOUT changes the limit)", timeout.Seconds())
	}
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			msg := strings.TrimSpace(errb.String())
			if msg == "" {
				msg = fmt.Sprintf("exited %d", exit.ExitCode())
			}
			lines := strings.Split(msg, "\n")
			return IndexerOutput{}, errors.New(lines[len(lines)-1])
		}
		return IndexerOutput{}, fmt.Errorf("couldn't run %s: %v", strings.Join(command, " "), err)
	}
	var o IndexerOutput
	if err := json.Unmarshal(out.Bytes(), &o); err != nil {
		return IndexerOutput{}, fmt.Errorf("its output isn't JSON: %v", err)
	}
	return o, nil
}

func DefinitionsOf(out IndexerOutput, pkg string) (map[string]Definition, bool) {
	p, ok := out.Packages[pkg]
	if !ok {
		return nil, false
	}
	defs := map[string]Definition{}
	for symbol, v := range p.Definitions {
		file, _ := v[0].(string)
		start, _ := v[1].(float64)
		end, _ := v[2].(float64)
		defs[symbol] = Definition{Symbol: symbol, File: file, StartLine: int(start), EndLine: int(end)}
	}
	return defs, true
}

func firstLine(s string) string { return strings.SplitN(strings.TrimSpace(s), "\n", 2)[0] }
