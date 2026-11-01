package app

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/aymanbagabas/go-osc52/v2"

	"github.com/hypertheory-labs/phoropter/tui/substrate"
)

// Links out of the TUI (nainsook): y copies the address of what you're on, Y
// the reference a plan's where: would use, and b opens the address in the
// browser.

// A linking screen says what "here" is. The reference is "" when there's
// none, as for a draft's entries before the draft lands.
type linking interface {
	here() (url, ref string)
}

func (m *Model) link(k string) {
	l, ok := m.top().(linking)
	if !ok {
		return
	}
	u, ref := l.here()
	u = linkable(u)
	switch k {
	case "y":
		m.status = copied(m.clip(u), "copied "+u)
	case "Y":
		if ref == "" {
			m.status = "nothing here has a reference: open a plan for its entries', and a draft's come when it lands"
			return
		}
		m.status = copied(m.clip(ref), "copied "+ref)
	case "b":
		if err := m.browse(u); err != nil {
			m.status = "couldn't open a browser: " + err.Error()
		} else {
			m.status = "opened " + u
		}
	}
}

func copied(err error, ok string) string {
	if err != nil {
		return "couldn't copy: " + err.Error()
	}
	return ok
}

// linkable is an address a browser opens as the page: a site on disk serves
// no index, so a directory gets its index.html, before the fragment.
func linkable(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "file" || !strings.HasSuffix(u.Path, "/") {
		return raw
	}
	u.Path += "index.html"
	return u.String()
}

// clipboard writes text with OSC 52, which the terminal turns into a clipboard
// write (Ghostty, tmux, ssh), and with the system's own command when there is
// one, since some terminals ignore OSC 52 silently. It fails only when neither
// could be tried.
func clipboard(text string) error {
	seq := osc52.New(text)
	if os.Getenv("TMUX") != "" {
		seq = seq.Tmux()
	}
	_, oscErr := seq.WriteTo(os.Stderr)
	var cmds [][]string
	switch runtime.GOOS {
	case "darwin":
		cmds = [][]string{{"pbcopy"}}
	case "windows":
		cmds = [][]string{{"clip"}}
	default:
		cmds = [][]string{{"wl-copy"}, {"xclip", "-selection", "clipboard"}, {"xsel", "--clipboard", "--input"}}
	}
	for _, c := range cmds {
		if _, err := exec.LookPath(c[0]); err != nil {
			continue
		}
		cmd := exec.Command(c[0], c[1:]...)
		cmd.Stdin = strings.NewReader(text)
		if cmd.Run() == nil {
			return nil
		}
	}
	if oscErr != nil {
		return errors.New("no clipboard command, and the terminal couldn't be written to")
	}
	return nil
}

func browser(u string) error {
	open := map[string][]string{"darwin": {"open"}, "windows": {"rundll32", "url.dll,FileProtocolHandler"}}[runtime.GOOS]
	if open == nil {
		open = []string{"xdg-open"}
	}
	return exec.Command(open[0], append(open[1:], u)...).Start()
}

// reference is how a plan's where: names an entry: the plan's file, then the
// entry's title, without its kind.
func reference(d *substrate.Doc, e *substrate.Entry) string {
	if d.Draft != nil || d.File == "" {
		return ""
	}
	if e == nil {
		return d.File
	}
	return fmt.Sprintf("%s › %s", d.File, e.Title)
}

func (s *listScreen) here() (string, string) {
	if s.cur == 0 {
		if s.draft == nil {
			return s.home.URL, ""
		}
		return s.home.DraftURL, ""
	}
	// The home page doesn't say a plan's file, and the TUI doesn't guess it
	// from the name and date.
	return s.plan(s.cur - 1).URL, ""
}

func (s *docScreen) here() (string, string) {
	r := s.rows[s.cur]
	switch {
	case r.entry != nil:
		return r.doc.URL + "#" + r.entry.ID, reference(r.doc, r.entry)
	case r.prompt != nil && r.prompt.After != "":
		return r.doc.URL + "#" + r.prompt.After, ""
	case r.note != nil && r.note.After != "":
		return r.doc.URL + "#" + r.note.After, ""
	}
	return r.doc.URL, reference(r.doc, nil)
}

// A place's reference is what an anchor at it says: its symbol, or its file.
func (s *placeScreen) here() (string, string) {
	if s.place.Symbol != "" {
		return s.place.URL, s.place.Symbol
	}
	return s.place.URL, s.place.File
}
