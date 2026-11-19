// phoropter-index-go: phoropter's indexer for Go (seersucker). It runs scip-go
// on each Go module the anchors ask for, and prints the definitions as
// phoropter's index JSON. See indexers/README.md for the contract.
//
//	phoropter-index-go --describe
//	phoropter-index-go <dir> [module path...]
//
// A module is found by its go.mod's module path, which is also the package
// field of its symbols: scip-go gomod phoropter-tui . `phoropter-tui/internal/tui`/docScreen#annotate().
package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/scip-code/scip/bindings/go/scip"
	"google.golang.org/protobuf/proto"
)

const version = "0.1.0"

type pkgOut struct {
	Definitions map[string][3]any `json:"definitions"`
}

type out struct {
	Indexer  string            `json:"indexer"`
	Version  string            `json:"version"`
	Scheme   string            `json:"scheme"`
	Packages map[string]pkgOut `json:"packages"`
	Missing  []string          `json:"missing"`
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--describe" {
		describe := map[string]any{"indexer": "phoropter-index-go", "version": version, "scheme": "scip-go", "extensions": []string{".go"}}
		if lookScipGo() == "" {
			describe["needs"] = []map[string]string{{"what": "scip-go", "install": installScipGo}}
		}
		emit(describe)
		return
	}
	if len(os.Args) < 2 {
		fail("usage: phoropter-index-go <dir> [module path...]")
	}
	dir, err := filepath.Abs(os.Args[1])
	if err != nil {
		fail(err.Error())
	}
	scipGo := findScipGo()
	modules := findModules(dir)
	asked := os.Args[2:]
	if len(asked) == 0 {
		for m := range modules {
			asked = append(asked, m)
		}
	}
	scipGoVer := scipGoVersion(scipGo)
	res := out{Indexer: "phoropter-index-go", Version: version + " (scip-go " + scipGoVer + ")", Scheme: "scip-go", Packages: map[string]pkgOut{}, Missing: []string{}}
	for _, mod := range asked {
		modDir, ok := modules[mod]
		if !ok {
			res.Missing = append(res.Missing, mod)
			continue
		}
		defs, err := index(scipGo, scipGoVer, dir, modDir, mod)
		if err != nil {
			fail(fmt.Sprintf("scip-go failed on %s: %v", mod, err))
		}
		res.Packages[mod] = pkgOut{Definitions: defs}
	}
	emit(res)
}

// findModules maps each go.mod's module path to its directory.
func findModules(dir string) map[string]string {
	modules := map[string]string{}
	moduleRE := regexp.MustCompile(`(?m)^module\s+(\S+)`)
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && (d.Name() == ".git" || d.Name() == "node_modules" || d.Name() == "vendor") {
			return filepath.SkipDir
		}
		if d.Name() == "go.mod" {
			if b, err := os.ReadFile(path); err == nil {
				if m := moduleRE.FindSubmatch(b); m != nil {
					modules[string(m[1])] = filepath.Dir(path)
				}
			}
		}
		return nil
	})
	return modules
}

// index runs scip-go on one module and keeps its definitions: the symbol with
// its version written as ".", the file relative to dir, and the lines.
func index(scipGo, scipGoVer, dir, modDir, mod string) (map[string][3]any, error) {
	tmp, err := os.CreateTemp("", "phoropter-index-go-*.scip")
	if err != nil {
		return nil, err
	}
	tmp.Close()
	defer os.Remove(tmp.Name())
	// scip-go 0.2 has an index command and skips tests itself; 0.1 took flags
	// alone and needed the repository's root.
	args := []string{"index", "-q", "-o", tmp.Name(), "--module-root=.", "--skip-tests"}
	if strings.HasPrefix(scipGoVer, "0.1.") {
		rel, _ := filepath.Rel(modDir, dir)
		args = []string{"-q", "-o", tmp.Name(), "--module-root=.", "--repository-root=" + rel}
	}
	cmd := exec.Command(scipGo, args...)
	cmd.Dir = modDir
	if b, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("%v: %s", err, strings.TrimSpace(string(b)))
	}
	b, err := os.ReadFile(tmp.Name())
	if err != nil {
		return nil, err
	}
	var idx scip.Index
	if err := proto.Unmarshal(b, &idx); err != nil {
		return nil, err
	}
	prefix, _ := filepath.Rel(dir, modDir)
	defs := map[string][3]any{}
	for _, doc := range idx.Documents {
		file := filepath.ToSlash(filepath.Join(prefix, doc.RelativePath))
		// Go's synthetic test packages point into its build cache, outside
		// the repository; only files in the repository are definitions here.
		if strings.HasPrefix(file, "../") || filepath.IsAbs(doc.RelativePath) {
			continue
		}
		ends := declEnds(filepath.Join(modDir, doc.RelativePath))
		for _, occ := range doc.Occurrences {
			if occ.SymbolRoles&int32(scip.SymbolRole_Definition) == 0 || strings.HasPrefix(occ.Symbol, "local ") {
				continue
			}
			parts := strings.SplitN(occ.Symbol, " ", 5)
			if len(parts) < 5 || parts[2] != mod {
				continue
			}
			parts[3] = "."
			key := strings.Join(parts, " ")
			if _, seen := defs[key]; seen {
				continue
			}
			// SourceRange and EnclosingSourceRange read both the typed ranges
			// newer scip-go writes and the older repeated-int32 form.
			r, ok := occ.EnclosingSourceRange()
			if !ok {
				if r, ok = occ.SourceRange(); !ok {
					continue
				}
			}
			start, end := r.Start.Line+1, r.End.Line+1
			// scip-go gives a definition's name, not its extent; the parser
			// knows where the declaration ends.
			if end == start {
				if e, ok := ends[start]; ok {
					end = e
				}
			}
			defs[key] = [3]any{file, start, end}
		}
	}
	return defs, nil
}

// declEnds maps the line each declared name is on to the line its
// declaration ends: a function's closing brace, a type's, a field's line.
func declEnds(path string) map[int32]int32 {
	ends := map[int32]int32{}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
	if err != nil {
		return ends
	}
	line := func(p token.Pos) int32 { return int32(fset.Position(p).Line) }
	mark := func(name *ast.Ident, end token.Pos) {
		if name != nil {
			if _, seen := ends[line(name.Pos())]; !seen {
				ends[line(name.Pos())] = line(end)
			}
		}
	}
	ast.Inspect(f, func(n ast.Node) bool {
		switch d := n.(type) {
		case *ast.FuncDecl:
			mark(d.Name, d.End())
		case *ast.TypeSpec:
			mark(d.Name, d.End())
		case *ast.ValueSpec:
			for _, name := range d.Names {
				mark(name, d.End())
			}
		case *ast.Field:
			for _, name := range d.Names {
				mark(name, d.End())
			}
		}
		return true
	})
	return ends
}

const installScipGo = "go install github.com/scip-code/scip-go/cmd/scip-go@latest"

// lookScipGo is scip-go's path, or "" when it isn't installed. --describe
// asks it too, so phoropter doctor can say it's missing (grosgrain).
func lookScipGo() string {
	if p, err := exec.LookPath("scip-go"); err == nil {
		return p
	}
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		home, _ := os.UserHomeDir()
		gopath = filepath.Join(home, "go")
	}
	if p := filepath.Join(gopath, "bin", "scip-go"); fileExists(p) {
		return p
	}
	return ""
}

func findScipGo() string {
	p := lookScipGo()
	if p == "" {
		fail("scip-go isn't installed: " + installScipGo)
	}
	return p
}

func scipGoVersion(scipGo string) string {
	b, err := exec.Command(scipGo, "--version").CombinedOutput()
	if err != nil {
		return "unknown"
	}
	return strings.TrimPrefix(strings.TrimSpace(string(b)), "v")
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func emit(v any) {
	enc := json.NewEncoder(os.Stdout)
	if err := enc.Encode(v); err != nil {
		fail(err.Error())
	}
}

func fail(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}
