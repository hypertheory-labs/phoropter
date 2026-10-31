# Next

Where phoropter-tui stands and what comes after. What's done has a plan in `plans/`
saying why. What's next often waits on phoropter, and names its seed in the
root `shelf/`. Phoropter may decide an ask differently or decline it. Then the item
here changes, and a gap that stays open is a cost the TUI pays by design, like the two
hops to an answer in another plan.

## The rule that doesn't change

Reasons come from phoropter's pages. The TUI computes only what's about the local
machine: the file tree, whether a local file matches what a page says (a draft's
sha256, later a file's blob id), how the working tree has moved. Anything the TUI would
infer about *why* goes to phoropter instead, so the person and the agent read the same
picture. Gimlet's decline of a separate client app is the ground for this.

The TUI's one write is a draft's notes file, by `docs/annotating.md`. It never writes
the draft, and never the substrate.

## Done

- **Plan mode,** before plans existed here: walk a plan and its as-built, flip each
  anchor between before, as built and now, follow rels, and the gaps ledger.
- **The ledger going quiet:** selvage and osier, phoropter's plans for saying in markup
  what the pages said in prose, closed seven gaps. Their workarounds are deleted, and
  `TestSelvageGapsStayClosed` guards them.
- **ramie** (reading the draft): a draft is recognized by its header, and its prompts
  and notes follow their entries. `r` reads a page again.
- **batiste** (answering from the TUI): `a` answers a prompt in the way its kind asks,
  notes an entry, or on About notes the whole draft. `s` skips on purpose, and `ctrl+e`
  hands the text to `$EDITOR`.
- **crewel** (what Jeff's first answers showed): a Select's choices are shown, its note
  is kept out of the list, and `esc` asks before throwing away what's typed.
- **challis** (the plan list reads like the page) and **voile** (lines on any theme,
  plans in boxes): a draft slot at the top, titles over names, newest first with `o`, a
  draft accent, theme colors. Plus `tab`/`shift+tab` through lists, with no plan.
- **nankeen** (landed plans aren't edited from the TUI), a decline.
- **dowlas** (the TUI moves in, as `tui/`): one repository, so plans on both sides link.
- **tarlatan** (drafts/ holds work in progress) and **organza** (a note on a landed plan
  says it matters now): `a` on a landed plan writes a note in `drafts/`, which the dev
  server shows on the plan's page. A revisit, a citation, or a reply answers it.

Guesses still waiting on use: batiste's (annotations.nvim and the TUI share one notes
file, once nvim answers prompts), challis's (a plan is found faster by title), and
voile's (nothing disappears on Ghostty's default theme).

## 1. What's left in the ledger

Rebuild `testdata/` when phoropter's profile changes (`node src/build.ts <repo> <commit>
--out …`, from the repository root). For each gap it closes, read the new property, delete
the workaround and the catalog entry, and add it to a test that keeps it closed.

- **Open, waiting on `›` paths resolving:** `where-path-unresolved`,
  `new-never-resolved`, `view-no-code`. `shelf/paths.md`.
- **Open, waiting on file pages:** `symbol-file-unlinked`, `place-url-convention`.
  `shelf/file-pages.md`.
- **Open, from drafts:** `answer-choices-unprofiled`. A Select's choices are only in
  `checked`. `shelf/select-choices.md`.
- **By design:** `same-as-deref` (marl kept a collapsed view whole) and
  `answer-elsewhere` (sorrel declined a skipping link). They stay, saying who decided.

## 2. File mode

*Needs:* ranges on file pages, symbol anchors on their file's page, a declared place
lookup, the blob id (`shelf/file-pages.md`). Without these the mode can be built, but only by
fetching every plan, which is a workaround the ledger would have to note.

- **Entry:** `phoropter-tui files [path]`, or a key from the plan list. The repo root is
  found by walking up to `.git` (as answering already does). `path` (say `src/`)
  filters the tree and isn't the root, since the substrate's paths are repo-relative.
- **Tree:** `git ls-files` for the list, so `.gitignore` is respected. This reads the
  index, not history. Files with anchors are lit, with an entry count, and the rest are
  dim. A toggle shows only the lit ones, the useful default for "what were they
  thinking."
- **Opening a file:** compute its git blob hash locally (sha1 of `blob <len>\0` plus the
  content) and compare it with the file page's. When they match, draw a gutter of bars
  colored by entry kind beside each anchored range, stacked where entries overlap. When
  they don't, say the file has changed since the site was generated and place nothing.
  Show the anchors as a list instead.
- **Cursor:** the right pane lists the entries covering the current line, newest first,
  with the reason. `enter` opens plan mode at that entry, so the two modes are one tool.
  `]`/`[` jump between anchored regions, and a fold hides everything else.
- **Can't be placed:** anchors whose quote isn't found at the generated commit, or that
  are new and unresolved, go in a section at the top of the file, not dropped. That's
  drift.

## 3. What's already at a draft's places

*Needs:* phoropter's ranking (`shelf/draft-places.md`). Loden put it on the shelf until "there's
a draft to try it on". This repo has written three plans as drafts since (ramie, batiste, challis), so that
condition is met.

For each anchor in a draft, the other entries already at that place, in phoropter's
order: unanswered guesses, declines, decisions this would change, drift. When an entry
would re-decide an earlier plan's ground, the TUI says what the guide says: that's a
revisit. It's rendered from phoropter's page, not ranked here, since ranking is
reasoning.

## 4. `init`, step 0

Create `plans/` and put up a first draft whose prompts are the defaults, each fallback
being the default. Phoropter can't write files, but the TUI can. Among its questions,
from Jeff: whether `drafts/` is ignored in git (yes for a pair on main, no for a team
that shares drafts on branches), and maybe the folder's name, if a repository already
has a `drafts/`. It waits until the guide's shape is decided.

## 5. `why`

`phoropter-tui why <path>:<line>` prints, as plain text, the entries about that line
with their reasons and sources, and the ones that can't be placed. It's for the person
in a terminal, and it's also a check on the rule above: if `why` can't answer from the
pages alone, an agent reading the HTML can't either, and that's a gap for the ledger.

## 6. Reasons to re-check

*Needs:* whatever phoropter decides on `shelf/re-check.md` (a page gathering unconfirmed
sources, age, drift, and open guesses; maybe `Holds while:`).

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
  names (`scip-go gomod …`). A missing indexer is a view status, not a failure. This
  repo, in Go, is the first case: its plans use file anchors until then.
- **The gaps ledger could be the conformance test for any UI:** one that needs nothing
  outside the profile is built correctly, and so is the API.

## Not planned

- **Git history in the TUI.** If something belongs in a reader's picture, it belongs in
  the substrate.
- **The LSP adapter, here.** If it happens, `internal/microdata` and `internal/substrate`
  are written to be reused by it, and file mode's data needs are the same as its own.
- **Settings.** Nothing is remembered between runs yet. Challis kept it that way: the
  longer it waits, the better the TUI has to be without them.
