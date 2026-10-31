package substrate

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/net/html"

	"phoropter-tui/internal/microdata"
)

// Client fetches substrate pages over http(s), or from a built site on disk
// through file:// URLs, and caches them: a site is one commit and never
// changes underneath us.
type Client struct {
	Root   *url.URL
	Ledger *Ledger
	HTTP   *http.Client

	mu    sync.Mutex
	pages map[string]*microdata.Document
}

// NewClient takes a URL, or a directory holding a built site.
func NewClient(root string, ledger *Ledger) (*Client, error) {
	if !strings.Contains(root, "://") {
		abs, err := filepath.Abs(root)
		if err != nil {
			return nil, err
		}
		root = (&url.URL{Scheme: "file", Path: filepath.ToSlash(abs)}).String()
	}
	if !strings.HasSuffix(root, "/") {
		root += "/"
	}
	u, err := url.Parse(root)
	if err != nil {
		return nil, err
	}
	return &Client{Root: u, Ledger: ledger, HTTP: http.DefaultClient, pages: map[string]*microdata.Document{}}, nil
}

// Resolve makes ref absolute against the site root.
func (c *Client) Resolve(ref string) string {
	u, err := c.Root.Parse(ref)
	if err != nil {
		return ref
	}
	return u.String()
}

// Fetch returns the parsed page at u. The fragment is ignored.
func (c *Client) Fetch(raw string) (*microdata.Document, error) {
	u, err := url.Parse(c.Resolve(raw))
	if err != nil {
		return nil, err
	}
	u.Fragment = ""
	key := u.String()
	c.mu.Lock()
	if d, ok := c.pages[key]; ok {
		c.mu.Unlock()
		return d, nil
	}
	c.mu.Unlock()

	body, final, err := c.get(u)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	root, err := html.Parse(body)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", key, err)
	}
	d := microdata.Parse(root, final)
	c.mu.Lock()
	c.pages[key] = d
	c.mu.Unlock()
	return d, nil
}

// Forget drops a page from the cache, so the next fetch reads it again.
func (c *Client) Forget(raw string) {
	u, err := url.Parse(c.Resolve(raw))
	if err != nil {
		return
	}
	u.Fragment = ""
	c.mu.Lock()
	delete(c.pages, u.String())
	c.mu.Unlock()
}

func (c *Client) get(u *url.URL) (io.ReadCloser, *url.URL, error) {
	if u.Scheme == "file" {
		p := u.Path
		if strings.HasSuffix(p, "/") {
			p += "index.html"
		} else if st, err := os.Stat(p); err == nil && st.IsDir() {
			// What a server's redirect to the slash URL would do.
			u2 := *u
			u2.Path += "/"
			u, p = &u2, p+"/index.html"
		}
		f, err := os.Open(p)
		if err != nil {
			return nil, nil, fmt.Errorf("not found: %s", u)
		}
		return f, u, nil
	}
	res, err := c.HTTP.Get(u.String())
	if err != nil {
		return nil, nil, err
	}
	if res.StatusCode != http.StatusOK {
		res.Body.Close()
		return nil, nil, fmt.Errorf("%s: %s", u, res.Status)
	}
	return res.Body, res.Request.URL, nil
}
