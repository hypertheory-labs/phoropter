package app

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// organdy: the TUI reads only what phoropter serves. It ships in the same
// binary, so this test is the boundary: no package under tui/ may import one
// of phoropter's own. It reaches phoropter over HTTP, like any other UI.
func TestTUIImportsNothingOfPhoropters(t *testing.T) {
	const own = "github.com/hypertheory-labs/phoropter/"
	filepath.Walk("..", func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(p, ".go") || strings.Contains(p, "testdata") {
			return nil
		}
		f, err := parser.ParseFile(token.NewFileSet(), p, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range f.Imports {
			path, _ := strconv.Unquote(imp.Path.Value)
			if strings.HasPrefix(path, own) && !strings.HasPrefix(path, own+"tui/") {
				t.Errorf("%s imports %s: the TUI may only reach phoropter through the pages it serves", p, path)
			}
		}
		return nil
	})
}
