# For phoropter: what a second reader found

phoropter-tui reads phoropter's pages and nothing else: no git, no source. Gimlet, the
plan for phoropter's first version, declined a separate client app because "a second
representation would let the person and the agent see different things," and said a
client "can come later as one more projection of the same pages." This is that
projection, and this is what it ran into.

Every time the TUI reads past the profile (parses prose, leans on a class name, or has
nothing to show), it notes a gap. `phoropter-tui gaps <site>` crawls a site and prints
them. The counts here are from mock-control at `f02528f` (4 plans) and phoropter at
`e8107f3` (7 plans), where the brief was first written, except where an item says otherwise. The current sites are in this repo's `testdata/`, built by phoropter's own
`build.ts`.

Nothing here is a format decision. Each item says what a reader wanted to ask, what the
pages say now, and which earlier plan's ground it stands on, so phoropter's plans can
decide it in their own frame. Some of these reopen a decline; those are revisits, not
tidy-ups.

The TUI's design rule, which shapes every ask: **anything about reasons belongs in
phoropter's pages.** The TUI computes only facts about the local machine (the file tree,
whether a local file matches the generated commit). If the TUI derived reasoning the
HTML lacks, the person would see more than the agent.

## 1. Gaps in what's already served

Ordered by how much they cost a reader.

### Closed by selvage and osier

Selvage (say in markup what the pages already say in prose) and osier (line numbers out
of the code's text) closed seven gaps. phoropter-tui reads each one from a property,
with its workaround deleted, on fixtures rebuilt at `1a83dbc`:

- `match-line-prose`: each view's `match` items give the lines. Pumice's README view
  reads `startLine` 94, as its note says.
- `line-numbers-unprofiled`: `codeStart` numbers an excerpt, and the `code` property's
  text is only the lines. The TUI reads it as plain microdata text, with no class names.
- `home-as-built-unmarked`, `home-revisits-unmarked`, `home-entries-prose`: the home
  list's `asBuilt`, `revisits` and `entryCount`.
- `new-two-shapes`: one `<meta>`, everywhere.
- `base-not-commit`: the profile says a `base` is text when it isn't a commit here.

Selvage's guess, restated in osier ("none of the closed gaps need prose or class
names"): **held**, at `1a83dbc`. At selvage's head it was partly disproved, since line
numbers were still inside `code`. Osier's work closed that last one.
`TestSelvageGapsStayClosed` keeps them closed.

### Still open: the three that need a `›` path to resolve

Selvage left these on the shelf, since resolving a path is more than markup. They
stand together: the second makes the other two possible.

### `where (new)` anchors never resolve, even where the work made them

*Reader question:* what did the work actually produce here?
*Now:* "made by the work, so there is nothing to resolve," with no views at any commit.
At the as-built head the thing exists, and it's the code a reader most wants.
*Seen:* 7 in mock-control (`new-never-resolved`); every one of pumice's README sections.
*Ask:* resolve a new anchor at the as-built head and the generated commit, by its `›`
path, since it has no quote. Lanyard's "nothing to resolve" is true at the base only.

### A `›` path resolves to nothing; only the quote does

*Reader question:* what range is "README.md › Setup › 1. Install"?
*Now:* the quote's line is found; the heading or key path is display only.
*Seen:* 16 anchors in mock-control (`where-path-unresolved`).
*Ask:* resolve heading paths (markdown) and key paths (JSON) to a line range. This is
what makes the item above possible, and the file-mode asks below.

### Views with no code

*Reader question:* what's there now, when the quote isn't?
*Now:* a file anchor whose quote isn't found at a commit has an empty view. A symbol
anchor falls back to the declaration ("so this is the declaration"); a file anchor has
nothing to fall back to.
*Seen:* 7 in mock-control, 29 in phoropter (`view-no-code`).
*Ask:* with path resolution, fall back to the `›` range, the way a symbol falls back to
its declaration.

### Not a gap: answers in another plan

Pumice's guess about Tailwind's `@source` is answered in vellum, two hops away. Sorrel
declined a skipping link: it "would say vellum answered pumice's guess, which only
vellum's prose says." The TUI follows the two hops, and the ledger notes it only as a
cost a reader pays, by design.

## 2. File mode: the code, with its reasons beside it

The TUI runs inside the repo, so it has the file tree. Opening a file should show, line
by line, which entries are about it, like `git blame`, but annotated with reasons
instead of authors. Gimlet declined "the heat map, the LSP adapter" because none was
needed to walk one plan, and recorded them "so nobody adds them helpfully." Jeff is
asking for this one; the same data would serve the LSP adapter he's considering.

*Reader question:* I'm in this file; which plans are about which lines, and which can't
be placed anymore?

What the pages would need:

1. **Ranges on a file page.** Each anchoredBy item gets its view at the generated commit:
   `status`, `startLine`/`endLine`, the matched line. Today it has `where` only, so
   placing anchors means fetching every plan.
2. **Symbol anchors on their file's page.** `MockPanel#` lives in
   `projects/mock-control/src/lib/panel/mock-panel.ts`, but no file page exists for it,
   and the symbol page names the file without linking one (`symbol-file-unlinked`).
   A file page should list every anchor in the file, symbol or not.
3. **A declared way to find a place by path.** `files/<path>/` works, but the profile
   doesn't say so, and a 404 can't distinguish "no plans here" from "wrong URL"
   (`place-url-convention`). A URI template in the profile would do.
4. **The file's git blob id at the generated commit.** The TUI hashes the local file the
   way git does, with no git, and overlays lines only when they match. When they don't,
   it says the file has changed since the site was generated, rather than putting reasons
   on the wrong lines.
5. Optional: **a plain-text alternate** (`rel="alternate" type="text/plain"`) for the
   file at the generated commit, so a reader without a checkout can still see it whole.

Anchors that don't resolve at the generated commit (quote not found, or new) belong in
the page as "plans about this file that can't be placed," not dropped. That's drift, and
it's the most interesting thing a file page can show.

## 3. Dimensionality: what's already here, for the cart

When the pair drafts a plan in the cart, its anchors name places. For each one, the
question is "what else is already here, and does it matter to what we're about to do?"
That's the guide's revisit rule, made visible before the work instead of found halfway
through.

Not every plan at a place matters equally. Roughly in order:

- **An unanswered guess** (or one answered `not tested`): this work might be its test.
- **A decline**: someone already considered what you're about to do and turned it down.
  This is the thing least likely to be in git history; a decline leaves no commit.
- **A decision you're changing**: that's a revisit, not a step in this plan.
- **A drifted anchor**: the ground has moved since that plan.

This ranking is reasoning, so it belongs in phoropter's cart pages, where the agent reads
it too. The TUI will render it.

### Declines are mostly invisible from places

In phoropter's own plans, 16 of 17 declines have no anchor, so no place page can show
them. In mock-control, 2 of 3 are anchored `where (new)` at where the declined thing
would have gone; that works, and the guide could suggest it. A decline with no anchor
can still be found by an agent if there's a page to search.

Marl declined cross-plan pages ("open guesses, declines, sources") because "they come
after source and verdict exist." Both landed since: source in marl, verdict in burlap.
So the condition that decline set is met. One page listing every entry on the site, with
kind, title, source, verdict, plan, and place, as microdata, would let an agent ask "has
anyone turned down X?" in one fetch.

## 4. A reason that was good then: is it still?

"We did that because we had no support license for Redis" is a good reason that expires.
The pages already carry signals for which reasons to re-check:

- **The source.** `Claude, not asked` (11 in mock-control, 31 in phoropter) and
  `Claude, unanswered, so its fallback` (4 in phoropter) are reasons nobody confirmed.
  The guide says so: such a reason "stays labeled that way until someone confirms it."
  There's no way yet to confirm one except a revisit.
- **Age**, from the plan's `added` commit.
- **Drift**, from the anchor's status at the generated commit.
- **Guesses** that are unanswered or answered `not tested`.

A page, or a section of the cart's place view, that gathers these would answer "which
reasons here rest on something nobody has checked lately?"

A format idea, for phoropter to weigh against lanyard's "a reason squeezed into a field
gets shorter and worse": a Guess says what would disprove it (`**Disproved if:**`), but a
Decision or Decline whose reason rests on a fact about the world doesn't say what would
undo it. An optional prose line after the reason, something like `**Holds while:**` (we
have no Redis support license), would tell a later reader, or an agent, exactly what to
re-check, and would make the revisit's "what moved: the world" concrete. It's prose, like
`Disproved if`, not a field. It's a format change, so it would be a plan, with the guide
changing in the same work.
