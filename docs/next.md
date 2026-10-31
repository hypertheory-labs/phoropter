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

*Started:* the seven gaps selvage and osier closed are read from properties, with
fixtures at `1a83dbc` and `TestSelvageGapsStayClosed` guarding them. What's left is the
three that need path resolution, and section 2's file-page items. `same-as-deref`
stays in the ledger as a cost by design (marl), like `answer-elsewhere`.

- Rebuild `testdata/` from both repos at their new heads (`node src/build.ts <repo>
  <commit> --out …` from `../phoropter`), and keep the old fixtures until the new
  code passes on both.
- For each gap phoropter closes, read the new property, delete the workaround, and
  remove the gap from the catalog. A test asserts that crawling the new fixtures notes
  none of the closed gaps, so a regression in phoropter shows up here as a test failure.
- Gaps phoropter declines stay, with their summary saying which plan declined them and
  why, as `answer-elsewhere` does now.

## 1½. The draft (loden's edge)

*Done in ramie* (reading the draft): the draft is recognized by its header, titled as
the working tree, and listed first; prompts and notes follow their entries; `r`
re-reads. What's below the next paragraph is kept for the reasoning, and for answering
prompts, which is still to come.

Phoropter's dev server serves the repo's one draft, `plan.draft.md`, at `draft/`, and
every page links it with `rel="draft"` while it exists. The profile defines it: a
`draft` item with entries as on a plan, plus `prompt` items (Question, Select, Affirm,
each with its `fallback` and any `answer`) and `annotation` items from the notes file.
A prompt or annotation belongs to the entry it follows.

The TUI's code today reads a draft page as an ordinary plan. That's wrong in a way that
matters. It would show the header's commit as if the draft were committed (on a draft
page it's only where the anchors resolve), and it would drop the prompts. So:

- Recognize the draft from the `draft` header (`readAt`, `sha256`), not from the URL,
  and say "draft, not committed, read at …" where a plan shows its commit.
- Show each prompt after its entry, with its fallback and answer, and annotations the
  same way.
- When the plan list's page has `rel="draft"`, put the draft at the top of the list.
- Re-fetch on a key: the server re-reads the draft on every request.

**Answering prompts, without vim.** Phoropter's pages are read-only, and loden declined
annotating in the browser (Chromium-only). The TUI is local and can write a file. It
can answer a draft's `Question`, `Select` and `Affirm` into `plan.draft.notes.md`,
following the annotating agreement exactly: restate the prompt, `**Answer:**`, `[x]` for
selections, `**Skipped.**` on purpose. That's the first path for a team that doesn't use
annotations.nvim. It writes the notes file only, never `plan.draft.md` and never the
substrate. Jeff's idea: annotate inside the TUI, with a key to open the draft in
`$EDITOR`, so annotations.nvim still works there.

*Needs:* the annotating agreement restated for this repo (phoropter doesn't depend on
loom), phoropter's `**If unanswered:**` addition, the one-draft rule, and conversion,
which deletes the notes.

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

## 3½. `init`, step 0

From phoropter's shelf (`guide-shape.md`): create `plans/` and put up a first draft
whose prompts are the defaults, each fallback being the default. Phoropter can't write
files, but the TUI can. It waits until the guide's shape is decided.

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

## Later, to talk through: a phoropter CLI with plugins

From Jeff, 2026-10-31. A `phoropter` command in the style of `gh` or `kubectl`. Starting
it indexes what's missing, builds the site, and serves it. UIs and language indexers
are plugins found on `PATH` (`phoropter ui` runs `phoropter-ui`, `phoropter index` runs
`phoropter-index-go`, and so on). This TUI becomes one UI you could plug in. That an
hour of work against the profile alone gave a usable reader is the evidence that the
product is the API, not a UI.

- **The boundary holds:** a UI plugin reads the served pages over HTTP, the same ones an
  agent reads, and never phoropter's internals.
- **Indexing lives in phoropter**, dispatched by the SCIP scheme each anchor already
  names (`scip-go gomod …`). A missing indexer is a view status, not a failure.
- **The gaps ledger could be the conformance test for any UI:** one that needs nothing
  outside the profile is built correctly, and so is the API.

## Not planned

- **Git history in the TUI.** If something belongs in a reader's picture, it belongs in
  the substrate.
- **The LSP adapter, here.** If it happens, `internal/microdata` and `internal/substrate`
  are written to be reused by it, and file mode's data needs are the same as its own.
