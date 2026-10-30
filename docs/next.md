# Next, after phoropter answers the brief

What this repo does once phoropter has worked through `for-phoropter.md`. Some items
depend on a specific ask; they say which. Phoropter may decide an ask differently, or
decline it; then the item here changes, and a gap that stays open is a cost the TUI pays
by design, like the two hops to an answer in another plan.

## The rule that doesn't change

Reasons come from phoropter's pages. The TUI computes only what's about the local
machine: the file tree, whether a local file matches the generated commit, how the
working tree has moved. Anything the TUI would infer about *why* goes to phoropter
instead, so the person and the agent read the same picture. Gimlet's decline of a
separate client app is the ground for this.

## 1. Let the ledger go quiet

*Started:* selvage's six closed gaps are read from properties, with fixtures at
`4ffb7d8` and `TestSelvageGapsStayClosed` guarding them. What's left is the `span.ln`
half of `line-numbers-unprofiled`, the three that need path resolution, `same-as-deref`
(the profile allows it, so it may simply be the shape), and section 2's file-page
items.

- Rebuild `testdata/` from both repos at their new heads (`node src/build.ts <repo>
  <commit> --out …` from `../phoropter`), and keep the old fixtures until the new
  code passes on both.
- For each gap phoropter closes, read the new property, delete the workaround, and
  remove the gap from the catalog. A test asserts that crawling the new fixtures notes
  none of the closed gaps, so a regression in phoropter shows up here as a test failure.
- Gaps phoropter declines stay, with their summary saying which plan declined them and
  why, as `answer-elsewhere` does now.

## 2. File mode

*Needs:* ranges on file pages, symbol anchors on their file's page, a declared place
lookup, the blob id. Without these the mode can be built, but only by fetching every
plan, which is a workaround the ledger would have to note.

- **Entry:** `phoropter-tui files [path]`, or a key from the plan list. The repo root is
  found by walking up to `.git`; `path` (say `src/`) filters the tree and is not the
  root, since the substrate's paths are repo-relative.
- **Tree:** `git ls-files` for the list, so `.gitignore` is respected. This reads the
  index, not history; the reasons still come only from the substrate. Files with
  anchors are lit, with an entry count; the rest are dim. A toggle shows only the lit
  ones, which is the useful default for "what were they thinking."
- **Opening a file:** compute its git blob hash locally (sha1 of `blob <len>\0` plus
  the content) and compare it with the file page's. When they match, draw a gutter of
  bars colored by entry kind beside each anchored range, stacked where entries overlap.
  When they don't, say the file has changed since the site was generated and don't
  place anything; show the anchors as a list instead.
- **Cursor:** the right pane lists the entries covering the current line, newest first
  (pumice, then vellum, then tallow, on a region), with the reason. `enter` opens plan
  mode at that entry, so the two modes are one tool. `]`/`[` jump between anchored
  regions; a fold hides everything else.
- **Can't be placed:** anchors whose quote isn't found at the generated commit, or that
  are new and unresolved, go in a section at the top of the file, not dropped. That's
  drift.

## 3. Cart mode

*Needs:* phoropter's `/cart` pages, and the "what's already here" ranking living there.

- Render a proto-plan the way plan mode renders a plan, with one more pane per anchor:
  the other entries at that place, in phoropter's order (unanswered guesses, declines,
  decisions this would change, drift).
- When phoropter flags that an entry would re-decide an earlier plan's ground, show it
  as the guide puts it: that's a revisit.
- Read only. The cart's edge feature may write, but never the substrate, and the TUI's
  first version of cart mode doesn't need to.

## 4. `why`

`phoropter-tui why <path>:<line>` prints, as plain text, the entries about that line
with their reasons and sources, and the ones that can't be placed. It's for the person
in a terminal, and it's also a check on the rule above: if `why` can't answer from the
pages alone, an agent reading the HTML can't either, and that's a gap for the ledger.

## 5. Reasons to re-check

*Needs:* whatever phoropter decides on section 4 of the brief (a page gathering
unconfirmed sources, age, drift, and open guesses; maybe `Holds while:`).

A view, from the plan list or a file, of the entries whose reasons nobody has confirmed
lately: "Claude, not asked," unanswered fallbacks, guesses not tested, anchors that
drifted. Rendered from phoropter's page, not computed here.

## Not planned

- **Git history in the TUI.** If something belongs in a reader's picture, it belongs in
  the substrate.
- **The LSP adapter, here.** If it happens, `internal/microdata` and `internal/substrate`
  are written to be reused by it, and file mode's data needs are the same as its own.
