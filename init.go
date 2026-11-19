package main

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// The first plan's template (muslin): the conventions as Decisions, each
// with phoropter's default as its fallback. init fills in {{base}}.
//
//go:embed init.md
var initTemplate string

// pointer is the only thing anyone pastes. It names nothing that changes,
// so it can't go stale: the format comes from the binary, the conventions
// from the first plan.
const pointer = `## Plans (phoropter)

This repository records its decisions as plans. ` + "`phoropter guide`" + ` prints the
format. The first plan, "How we work here" (the oldest in ` + "`plans/`" + `, or
` + "`drafts/plan.draft.md`" + ` until it converts), says how we work. Read both before
other work. While the first plan is still a draft, offer to work through it with the
person before anything else.`

const draftPath = "drafts/plan.draft.md"

var errDraftExists = errors.New("a draft exists")

// writeFirstDraft writes the template as the repository's one draft, based
// at base. It never replaces a draft that's already there.
func writeFirstDraft(root, base string) error {
	path := filepath.Join(root, draftPath)
	if _, err := os.Stat(path); err == nil {
		return errDraftExists
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	text := strings.Replace(initTemplate, "{{base}}", base, 1)
	return os.WriteFile(path, []byte(text), 0o644)
}

// initWalkthrough is what the person reads after init: what happened, and
// the first round, step by step. It's printed by the binary, so it changes
// only when the binary does.
func initWalkthrough(root, url string, plans int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Wrote %s: this repository's first plan, as a draft.\n\n", draftPath)
	b.WriteString("It's about how you and your agent work here, not the code. Every entry is\n" +
		"phoropter's default. Answer a prompt or annotate anything you'd change; whatever\n" +
		"you leave alone converts as it is.\n")
	if plans > 0 {
		fmt.Fprintf(&b, "\nThis repository already has %s. The draft sits beside them as the\n"+
			"plan about how you work.\n", count(plans, "plan"))
	}

	b.WriteString("\n1. Tell your agent. Paste this into " + instructionsFile(root) + ":\n\n")
	for _, line := range strings.Split(pointer, "\n") {
		b.WriteString(strings.TrimRight("     "+line, " ") + "\n")
	}

	if url != "" {
		fmt.Fprintf(&b, "\n2. Read the draft at %sdraft/, or with phoropter ui.\n", url)
	} else {
		b.WriteString("\n2. Run phoropter, or ask your agent to. The draft is at draft/ on the site\n" +
			"   it serves, and phoropter ui reads it in the terminal.\n")
	}
	b.WriteString("\n3. Annotate what you'd change, then work the draft with your agent. When\n" +
		"   you're both confident, it converts into plans/, and that's your first plan.\n" +
		"   Silence means proceed. phoropter guide explains drafts and notes.\n")
	return b.String()
}

// instructionsFile says where the pointer would go: the agent instructions
// the repository already has, or the usual names when it has none. init
// never edits them (muslin).
func instructionsFile(root string) string {
	var found []string
	for _, name := range []string{"CLAUDE.md", "AGENTS.md"} {
		if _, err := os.Stat(filepath.Join(root, name)); err == nil {
			found = append(found, name)
		}
	}
	switch len(found) {
	case 0:
		return "your agent's instructions (CLAUDE.md or AGENTS.md)"
	case 1:
		return found[0]
	}
	return strings.Join(found, " and ")
}

func countPlans(root string) int {
	entries, _ := os.ReadDir(filepath.Join(root, "plans"))
	n := 0
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, ".md") && !strings.HasSuffix(name, ".as-built.md") {
			n++
		}
	}
	return n
}

func count(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return fmt.Sprintf("%d %ss", n, noun)
}
