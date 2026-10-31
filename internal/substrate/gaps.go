package substrate

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// A Gap is somewhere the TUI had to read past the profile: parse prose, lean on
// a class name or attribute the profile doesn't define, or go without
// something it wanted. The ledger is the experiment's other output: a list of
// what the hypermedia API is missing, found by using it.
type Gap struct {
	ID      string
	Summary string
	Count   int
	Example string // the first URL where it happened
	Detail  string // what was seen there
}

var catalog = map[string]string{
	"view-no-code":          "A view exists at this commit but carries no code, so this layer can't be shown.",
	"same-as-deref":         "A collapsed generated view omits its code, and the TUI copies it from the view sameAs names. Deliberate: marl kept the collapsed view whole (role, commit, status) and dropped only the code, so an agent reading three views gets three answers.",
	"new-never-resolved":    "A where (new) anchor has no views, even at the as-built head where the work made it.",
	"where-path-unresolved": "A where with a › path (heading or key) is resolved only by its quote; the path gives no range.",
	"symbol-file-unlinked":  "A symbol page names its file but doesn't link a file page, so \"other plans in this file\" misses it.",
	"answer-elsewhere":      "A guess answered in another plan is two hops away (anchoredBy, then answeredBy). Deliberate: sorrel declined a skipping link, since only the answering plan's prose says it answers this one.",
	"place-url-convention":  "There's no declared way to find a place by path; files/<path>/ is a convention the profile doesn't state.",
}

type Ledger struct {
	mu   sync.Mutex
	gaps map[string]*Gap
}

func NewLedger() *Ledger { return &Ledger{gaps: map[string]*Gap{}} }

// Note records one occurrence. A nil ledger ignores it.
func (l *Ledger) Note(id, where string, detail string, args ...any) {
	if l == nil {
		return
	}
	summary, ok := catalog[id]
	if !ok {
		panic("unknown gap " + id)
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	g := l.gaps[id]
	if g == nil {
		g = &Gap{ID: id, Summary: summary, Example: where, Detail: fmt.Sprintf(detail, args...)}
		l.gaps[id] = g
	}
	g.Count++
}

// Gaps are sorted most frequent first.
func (l *Ledger) Gaps() []Gap {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]Gap, 0, len(l.gaps))
	for _, g := range l.gaps {
		out = append(out, *g)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func (l *Ledger) Report() string {
	var b strings.Builder
	for _, g := range l.Gaps() {
		fmt.Fprintf(&b, "%-24s ×%-3d %s\n", g.ID, g.Count, g.Summary)
		fmt.Fprintf(&b, "%-29s e.g. %s\n", "", g.Example)
		if g.Detail != "" {
			fmt.Fprintf(&b, "%-29s %s\n", "", g.Detail)
		}
	}
	return b.String()
}
