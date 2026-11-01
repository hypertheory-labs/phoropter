# Seed: `init`, a first plan that holds the repository's conventions

From Jeff (2026-10-31): in a new repository, an `init` process creates a first plan that
holds as much of the guide as can move into a plan. The first round with an agent is
working that plan as a draft: annotating it, answering its prompts, converting it.

What came before it:
- `tui/docs/next.md` §4: `init` creates `plans/` and puts up a first draft whose prompts
  are the defaults, each fallback being the default.
- Jeff, on batik's draft: the first draft is about how the pair works together, e.g.
  "the agent always asks before converting a plan to code", defaulting to no.
- Jeff, on tarlatan's draft: whether `drafts/` is ignored in git is an `init` question,
  and maybe the folder's name, "a seam for configuration later".

The idea it suggests: the guide splits into two layers.
- **The format** stays in the guide: what a plan file looks like, which phoropter
  parses. It doesn't vary by repository.
- **The conventions** move into the first plan, as decisions with prompts and defaults:
  when to draft, who converts and when, committing the plan before the work, whether
  `drafts/` is committed, whether the agent asks before converting, silence means
  proceed, no veto. After that, a team's way of working is testimony like any other
  decision, and changing a convention is a revisit, with its reason. It would replace
  most of the README's "tell your agent" block: the agent reads its conventions from
  `plans/`, where the pair decided them.

To talk through:
- Is it `phoropter init`, a command that writes on purpose, as `phoropter index` does
  (batik), or does it live in a UI?
- Does phoropter read the first plan to change what it does, or is it only for the
  person and the agent?
- Item by item, where is the line between format and convention?

**From the course repository, 2026-11-01.** An agent learns that phoropter exists only
from something it reads unprompted, so this is three layers, each where it belongs:
- **a pointer** in `CLAUDE.md` or `AGENTS.md`, a line or two, which `init` can write or
  print;
- **the format**, from `phoropter guide` (no server needed);
- **the conventions**, in the first plan, including who starts the server.

That other Claude's "notes for phoropter" were a gaps file before it had the name. Jeff
suggested loom: loom's orientation is a general pointer, and adopting the guide as an
external gives gaps and usages a home. Loom's publication document
(`git.hypertheory-labs.dev/loom/bedrock`, `.loom/published/publication.md`) lines up:
- "a published one is a contract" is the line between format and conventions;
- "a document… published inside the thing that cannot be seen" is the guide served by
  the server it explains how to start;
- "a published document that quietly requires a private one" is the guide's citations
  of other repositories' plans.

The line to keep: phoropter publishes and loom adopts, so phoropter still works without
loom. `indexers/README.md` is a contract too (the roslyn indexer depends on it). Publishing
needs a fetch someone else performs, and phoropter's raw `guide.md` returned 404 on
2026-11-01, so the readers have to be named.

**Jeff, on the lens draft (percale), 2026-11-01:** the server could carry recipes for the
agent, like skills: when the person says "mark these for me", the agent has somewhere to
look up what that means. Converting a draft and answering notes are recipes too.
`phoropter guide` already shows the binary can carry agent-facing text with no server, so
`init` could install a skill file that points at it. And recipes let the agent act
unprompted: "I made a lens through which we might look at this."

**Jeff, reading percale:** the split between static pages (a commit's testimony) and what
the dev server adds at run time (the draft, notes, lenses) is the seam for recipes too. A
`tools/` (or `recipes/`) page the dev server serves, configured per repository by what
`init` wrote, would give the agent its "what does 'mark these' mean here" in the same
hypermedia it already reads, and a built site would never carry it.

**From Jeff's "today" lens (dimity's timeout, seersucker's discovery), 2026-11-01:** some of
what `init` sets up is tool settings, not conventions. The indexer timeout is really per
repository (a build-based .NET indexer needs more room than a Go one), but today it's an
environment variable. And which indexer answers when two handle a language (roslyn and
scip-dotnet, both on Jeff's machine) is the "per-repository preference" dimity put off
until someone needed it. Killing a timed-out indexer outright is behavior with a reason,
not a setting. The split: conventions in the first plan as decisions; settings in a
small file phoropter reads (under `.phoropter/`, beside the index store), each with a
plan decision saying why it's set that way. The file holds the value, and the plan
holds the reason.
