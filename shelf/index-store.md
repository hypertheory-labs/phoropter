# Seed: indexes are facts about commits, kept apart from pages

A full plan, drafted with Jeff and answered, then shelved before any work: it waits
on the plan for the `phoropter` command with no arguments, since `phoropter index` is
where they meet. Restate it into a draft when that plan lands, with a new `base:` and a
name; everything below is as Jeff answered it.

---


# Indexes are facts about commits, kept apart from the pages built from them

The first of four plans toward a `phoropter` command. It comes first because the
command's other questions (what it does with no arguments, how indexers plug in, how it
ships) all lean on where indexes live. Jeff's first idea was to commit the built site and
rebuild only what changed since. Looking at why gimlet didn't commit the site turned it
into this: the site was never the expensive part. The index is.

Today an index lives in memory for one build. Every build and every server start
re-indexes every commit an anchor needs, from scratch, with the toolchain installed
today. For mock-control that's a couple of seconds. The real cost is durability: a
symbol anchor on a plan from two years ago needs that commit indexed, and that era's
toolchain and modules may not exist anymore.

So phoropter gets three layers:
- **Plans:** testimony, committed, never edited.
- **Indexes:** facts about commits. They never change, so they can be stored.
- **Pages:** a projection of the two. Cheap to rebuild, never committed.

## Decline: Committing the built site and rebuilding what changed
- where: `plans/2026-10-30-gimlet.md › The generated site isn't committed`

**Because (gimlet, then Claude, measured):** gimlet's reasons hold: the site is
reproducible from the commit, and generated output in git would bury the plans in diff
noise. Two more came up weighing it:
- **A committed site is always one commit behind.** A site built from commit N can't be
  inside commit N. Committing it makes N+1, which the site doesn't describe.
- **Nearly every page changes on every commit anyway.** A plan's page shows each anchor
  at its base, its as-built head, and now, and "now" moves with every commit. So do the
  home page and the place pages. Rebuilding what changed would rebuild almost all of it.

Jeff's instinct, to keep an artifact and do only the missing work, was right about a
different artifact: the index.

## Decision: An index is stored per commit, package and indexer, and never changes
- where: `src/scip.ts`
- quote: `export function indexCommit(repo: Repo, sha: string, packages: Set<string>): Map<string, Definition> {`

**Because (Claude, not asked):** what an indexer says about a commit is fixed once the
commit is. The key is the commit's sha, the package (its name, as a SCIP symbol names
it), and the indexer with its version. A new indexer version is a new key, not an
overwrite. Phoropter stores only what it reads: each definition's version-less symbol,
file, and line range, as JSON. It doesn't store the raw `.scip` file, which holds
occurrences phoropter never uses.

## Decision: The indexes plans need are committed, in `.phoropter/index/`
- where: `.gitignore`
- quote: `site/`

**Because (Claude, not asked):** every plan anchors its base commit, and maybe an as-built
head. Those commits are fixed, finite, and grow only as plans land. Committing their
indexes means a plan's symbol anchors still resolve years from now, after the toolchain
that could index that commit is gone. It works the way a lockfile does: a recorded fact
that saves recomputing, and that a build can check. Files are
`.phoropter/index/<sha>/<package>.json`, each naming its indexer and version (Jeff).

It varies by branch, safely (Jeff asked). A branch's `.phoropter/index/` holds the
indexes its own plans' commits need. Files are keyed by sha and never change, so merging
two branches only adds directories, and two indexes of the same commit are identical
files. It also survives squash merges, which the guide warns can leave a plan's `base:`
naming a commit that's gone. The index keyed by that sha is still committed, so the
plan's anchors still resolve.

Weighed and set aside: a cache outside the repository only, where each machine indexes
for itself and an old anchor dies with its toolchain; and git notes, which attach data
to a commit without changing it. Notes would be the most literal fit, but they aren't
pushed or fetched without extra configuration, GitHub doesn't show them, rebases leave
them behind unless configured, and merging them takes a command few people know.

## Decision: The index for "now" lives in a local cache
- where: `plans/2026-10-30-gimlet.md › TypeScript, indexing with scip-typescript at the generated commit`

**Because (Claude, not asked):** the generated commit moves every time, so committing its
index would be exactly the churn gimlet avoided. It goes in a cache outside the
repository, under the same key, so serving the same commit twice indexes it once. Once
a plan lands with that commit as its base, the index moves from the cache to
`.phoropter/index/`.

## Decision: A build indexes only what's missing
- where: `src/site.ts`
- quote: `// Index each commit that a SCIP anchor needs, once per set of packages.`

**Because (Jeff):** "at startup you compared the offset: when the site was last built,
where we are now, index the missing pieces." Applied to indexes rather than pages: a
build works out which (commit, package) pairs its anchors need, reads what's stored, and
runs the indexer only for what isn't. What it indexes goes in the local cache.

Putting a plan commit's index into `.phoropter/index/` means writing into the
repository being read. Nothing in phoropter does that today. Gimlet made the site
read-only ("never a place to write"), and loden says it flatly: "Phoropter only reads."
So a build doesn't write there. An explicit command does, `phoropter index`, which the
first plan of the four (the command with no arguments) can run on purpose. A build says
which plan commits are only in the cache, so you know to run it. Letting any build write
there, as an install updates a lockfile, would be a revisit of loden, and isn't needed
yet (Jeff: "safest for now").

## Decision: A missing indexer is a view status, not a failed build
- where: `src/resolve.ts`
- quote: `status: 'found' | 'not-found' | 'no-quote' | 'no-file' | 'no-symbol';`

**Because (Claude, not asked):** a repository may anchor symbols in a language whose
indexer isn't installed on this machine. That anchor's view says `no-index` and names the
indexer it needed, and every other page builds. The profile defines the status. When
the second plan of the four makes indexers pluggable, this is how a missing plugin shows.

## Decision: The dev server's draft reads the same store
- where: `src/draft.ts`
- quote: `resolveEntries(cx, plan, plan, commits, packages, false);`

**Because (Claude, not asked):** the draft is re-read on every request, and its anchors
resolve at its base and at now. Both indexes come from the store, so a refresh doesn't
wait on an indexer. Indexes serve more than pages: the TUI's file mode, `why`, and an
LSP adapter would read the same facts. That's the reason for keeping them apart.

## Decline: Indexers other than scip-typescript, in this plan
**Because (Claude, not asked):** the key names its indexer, so the store is ready for
more. Choosing and finding them is the second plan's job: the plugin contract.

## Guess: A plan's symbol anchors resolve from the store alone
**Disproved if:** with scip-typescript uninstalled, building mock-control from a clone
whose `.phoropter/index/` holds its plans' commits leaves a plan's symbol anchor at
`no-index` at its base or as-built head.
