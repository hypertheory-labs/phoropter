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
