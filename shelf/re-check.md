# Seed: reasons to re-check

From the TUI's brief (`tui/docs/for-phoropter.md`, section 4). "We did that because we
had no support license for Redis" is a good reason that expires. The pages already
carry signals: sources nobody confirmed (`Claude, not asked`; `Claude, unanswered, so
its fallback`), age from `added`, drift, and guesses unanswered or `not tested`. A page
gathering them would answer "which reasons here rest on something nobody has checked
lately?"

A format idea to weigh against lanyard's "a reason squeezed into a field gets shorter
and worse": an optional prose line after a reason, `**Holds while:**` (we have no Redis
support license), naming what would undo it, as `**Disproved if:**` does for a guess.
A format change is a plan, with the guide changing in the same work.
