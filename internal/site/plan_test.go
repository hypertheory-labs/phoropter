package site

import (
	"reflect"
	"strings"
	"testing"
)

func TestUnwrapCodeSpan(t *testing.T) {
	for in, want := range map[string]string{
		"`a`":              "a",
		"`` a`b ``":        "a`b",
		"`selector: 'x',`": "selector: 'x',",
		"not code":         "not code",
		// Lenient on purpose: awl wraps a title that contains backticks in
		// single backticks; phoropter reads the whole span.
		"`plans/2026-10-30-grommet.md › An as-built has `head:` in its frontmatter, the commit where the work landed`": "plans/2026-10-30-grommet.md › An as-built has `head:` in its frontmatter, the commit where the work landed",
	} {
		if got := UnwrapCodeSpan(in); got != want {
			t.Errorf("UnwrapCodeSpan(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestEntriesAnchorsAndQuotes(t *testing.T) {
	doc := ParsePlan("---\ncart: test\nbase: abc1234\n---\n\n# A plan\n\nIntro.\n\n## Decision: Do the thing\n- where: `src/a.ts › b`\n- quote: `const b = 1;`\n- where (new): `src/c.ts`\n\n**Because (Jeff):** reasons.\n\n- a list that is prose\n\n## Guess: It works\n**Disproved if:** it doesn't.\n\n## Chose: A thing\n## Answered: It works\n**Answer (measured):** partly disproved, on one run.\n\n## Answered: Unclear\n**Answer (measured):** mostly fine.\n\n## Answered: Capitalized\n**Answer (Jeff):** Held. Every time.\n")
	if !reflect.DeepEqual(doc.Frontmatter, []KV{{"cart", "test"}, {"base", "abc1234"}}) || doc.Title != "A plan" || doc.Intro != "Intro." {
		t.Fatalf("doc = %+v", doc)
	}
	var kinds, verdicts []string
	for _, e := range doc.Entries {
		kinds = append(kinds, e.Kind)
	}
	for _, e := range doc.Entries[3:] {
		verdicts = append(verdicts, e.Verdict)
	}
	if strings.Join(kinds, ",") != "Decision,Guess,Chose,Answered,Answered,Answered" || !reflect.DeepEqual(verdicts, []string{"partly disproved", "", "held"}) {
		t.Errorf("kinds %v verdicts %v", kinds, verdicts)
	}
	d, g := doc.Entries[0], doc.Entries[1]
	want := []Anchor{{Where: "src/a.ts › b", Quote: "const b = 1;", HasQuote: true}, {Where: "src/c.ts", IsNew: true}}
	if d.Slug != "do-the-thing" || !reflect.DeepEqual(d.Anchors, want) || !strings.Contains(d.Body, "a list that is prose") || d.Source != "Jeff" {
		t.Errorf("decision = %+v", d)
	}
	if g.Source != "" || g.Kind != "Guess" || len(g.Anchors) != 0 {
		t.Errorf("guess = %+v", g)
	}
}

func TestHeadingsInFencesArentEntries(t *testing.T) {
	doc := ParsePlan("# T\n\n## Decision: One\n```md\n## Decision: not an entry\n```\n\n## Decline: Two\n")
	if len(doc.Entries) != 2 || !strings.Contains(doc.Entries[0].Body, "not an entry") {
		t.Errorf("entries = %+v", doc.Entries)
	}
}
