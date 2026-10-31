---
base: 369f47e
---

# A sample draft, kept handy for trying the TUI

**Not a real plan.** Copy it in with `cp testdata/sample.draft.md plan.draft.md`, then
answer, note, skip and refresh. Take it down with `rm plan.draft.md
plan.draft.notes.md`. Nothing in it will be built. It has one prompt of each kind, an
anchor that resolves, one whose quote isn't there, a `where (new)`, and an entry with no
anchor.

## Decision: The dim line under each plan drops the date when the pane is narrow
- where: `internal/tui/screens.go`
- quote: `func meta(p substrate.PlanSummary) string {`

**Because (Claude, not asked):** the name and the as-built mark say more than the date,
and the date is in the detail pane anyway.

### Affirm: Should the entry count go before the date, too?
**If unanswered:** yes.

## Decision: Open prompts are counted in the plan list's slot, by kind
- where: `internal/tui/screens.go`
- quote: `func openPrompts(d *substrate.Doc) int {`

**Because (Claude, not asked):** "2 open prompts" doesn't say whether they're quick
yes-or-no questions or ones that need writing.

### Select: Which kinds are worth counting separately?
- Question
- Select
- Affirm

**If unanswered:** Question apart from the other two, since it's the one that takes
writing.

## Decision: Plans can sparkle when they're new
- where: `internal/tui/screens.go`
- quote: `func sparkle(p substrate.PlanSummary) bool {`

**Because (Claude, not asked):** this quote isn't in the code at the base commit, so this
anchor shows how a not-found looks.

## Decision: A help overlay for drafts gets its own file
- where (new): `internal/tui/drafthelp.go`

**Because (Claude, not asked):** this file doesn't exist, so this anchor shows how a
`where (new)` looks: made by the work, with nothing to resolve.

## Decline: Sound effects
**Because (Claude, not asked):** no anchor here, on purpose: a decline about something
that was never built has no code to point at.

## Guess: Nobody misses a mouse
**Disproved if:** after a week of use, someone reaches for the trackpad to pick a row.

### Question: What would you click first, if you could?
**If unanswered:** nothing; keys stay the only way in.
