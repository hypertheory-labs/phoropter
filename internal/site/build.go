package site

import (
	_ "embed"
	"os"
	"path/filepath"
)

//go:embed style.css
var styleCSS []byte

// Build writes the site for one repo at one commit into out.
func Build(repo *Repo, ref, out string, store *Store, guide Guide) (*Site, int, error) {
	s, err := BuildSite(repo, ref, store)
	if err != nil {
		return nil, 0, err
	}
	pages := RenderSite(s, guide)
	os.RemoveAll(out)
	for _, p := range pages {
		file := filepath.Join(out, p.Href, "index.html")
		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			return nil, 0, err
		}
		if err := os.WriteFile(file, []byte(p.HTML), 0o644); err != nil {
			return nil, 0, err
		}
	}
	if err := os.WriteFile(filepath.Join(out, "style.css"), styleCSS, 0o644); err != nil {
		return nil, 0, err
	}
	return s, len(pages), nil
}
