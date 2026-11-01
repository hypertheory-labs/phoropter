# Seed: `init`, how a repository starts working with plans

From Jeff (2026-10-31): in a new repository, an `init` process creates a first plan holding
as much of the way of working as can move into a plan, and the first round with an agent
is working that plan as a draft. It has been put off on purpose. Each day of real use
has found another layer, and this seed collects them as they stand.

## The layers, each where it belongs

An agent learns phoropter exists only from something it reads unprompted, and everything
after that should come from somewhere that can't go stale.

1. **A pointer,** a line or two in `CLAUDE.md` or `AGENTS.md`: "this repository records
   decisions as plans; run `phoropter guide`, and read the first plan for how we work."
   `init` writes it, or prints it to paste. Only the pointer is pasted, so only the pointer
   can go stale.
2. **The format,** from `phoropter guide`, built into the binary (since poplin). It's the
   same in every repository. It's a contract, in loom's sense: "a published one is a
   contract".
3. **The conventions,** in the first plan, as Decisions with prompts and defaults: when to
   draft, who converts and when, committing the plan before the work, whether `drafts/` is
   committed, whether the agent asks before converting, silence means proceed, no veto,
   who starts the server. After that, a team's way of working is testimony, and changing
   a convention is a revisit, with its reason.
4. **Guesses about the pair,** in that same first plan: "silence means proceed works for
   us", "we'll read drafts first". The conventions get tested, not assumed. The Guess is
   what keeps the format honest, and it should apply to the people as much as the code.
5. **Settings,** in a small file phoropter reads, under `.phoropter/` beside the index
   store: the indexer timeout (really per repository, today an environment variable), and
   which indexer answers when two handle a language (roslyn and scip-dotnet, both on
   Jeff's machine; dimity put this off until someone needed it). Each setting gets a
   Decision in a plan saying why. The file holds the value, and the plan holds the reason.
   Behavior with a reason (killing a timed-out indexer outright) isn't a setting. And
   nothing in it holds state: nothing tracks state in phoropter.
6. **Recipes for the agent,** served by the dev server's run-time layer: what "mark these"
   means here, how to convert a draft, how to answer notes. That layer already holds what
   isn't a commit's testimony (the draft, notes, lenses), and a built site never carries it.
   `init` could install a skill file that points at it. Recipes are what let the agent act
   unprompted: "I made a lens through which we might look at this."

## The receipt that makes this urgent

The README's "tell your agent" block has nearly doubled in a day (the server, lenses,
Declines gathered in a lens). The course repository's `CLAUDE.md` still holds the version
pasted before lenses existed, and nothing says it's old. A pasted block goes stale
silently. A pointer plus text from the binary stays current with the binary.

## How it might start

The proto-plan cycle (`shelf/lenses.md`) fits: `init` could put up the conventions as a
lens and a draft, the pair talks it through, and the draft converts into the first plan.
Or it's simpler: `init` writes the draft directly, with every prompt's fallback being the
default (`tui/docs/next.md` §4).

## Publication

The guide and `indexers/README.md` are contracts other repositories depend on, and the
roslyn indexer depends on the second. Loom's publication document
(`git.hypertheory-labs.dev/loom/bedrock`, `.loom/published/publication.md`) lines up:
- "a document… published inside the thing that cannot be seen" was the guide served only
  by the server it explains how to start, until `phoropter guide`;
- "a published document that quietly requires a private one" was the guide's inline
  citations, now moved to the end and called history (poplin).

Phoropter publishes and loom adopts, so phoropter still works without loom. A
publication is only finished when someone else can fetch it, and phoropter is private for
now ("public at some point", Jeff), so its readers have to be named until then. The other
Claude's "notes for phoropter" were a gaps file before it had the name. Adopting the
guide as a loom external gives gaps and usages a home.

## Open

- `phoropter init`, a command that writes on purpose, as `phoropter index` does (batik), or
  something a UI does?
- Does phoropter read the first plan to change what it does, or is it only for the person
  and the agent? Settings in a file suggest the second.
- Item by item, where is the line between format and convention?
- How much of the README's agent block survives once the pointer, the first plan and the
  recipes exist? Probably almost none.

## Where it came from

- Jeff, 2026-10-31: the idea. `tui/docs/next.md` §4: a first draft whose fallbacks are the
  defaults.
- Jeff, on batik's draft: the first draft is about how the pair works ("the agent always
  asks before converting", defaulting to no).
- Jeff, on tarlatan's draft: whether `drafts/` is ignored is an `init` question.
- The course repository, 2026-11-01: the pointer layer, and who starts the server. Jeff
  suggested loom, which brought in publication.
- Jeff, on percale's draft: recipes, like skills. Reading percale: the dev server's
  run-time layer is their seam.
- Jeff's "today" lens: settings, separate from conventions.
- 2026-11-01, later: lenses as proto-plans, nothing tracks state, and Guesses applied to the
  pair.
