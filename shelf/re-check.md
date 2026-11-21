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

Jeff, 2026-11-21, from the other side: "six months ago we guessed we wouldn't need this
because of X", and X turned out false. Nothing needs building, but the field of
opportunity shifted: "but you said you didn't want X!" now has an answer. The format
already has the shape: a revisit whose work is no code, saying what moved and that the
old Decline no longer binds. What's missing is the link from a Decline to the
condition it rests on, which is `**Holds while:**`. And Jeff's Redis framing: "we did a
bunch of stuff a particular way because we couldn't use Redis; if major work happens
here, look at it through that lens." Not a persistent lens (lenses stay ephemeral): a
persistent lens would be a photograph. What persists is the prescription, the
conditions written in the plans, and the lens is ground fresh from it when asked
("which entries hold while we can't use Redis?"). `Holds while:` lines are what
would make that lens findable by more than meaning.
