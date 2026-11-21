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

Same day, the many-reasons version (Jeff, then Claude): a stance many plans rest on
("we're AWS-first, for the Amazon relationship") wants re-evaluating as a whole when it
moves. Not a lens that plans get added to (maintained, so it drifts) and not a new kind
of ambient plan (a plan from three years ago isn't present). Loom's venues already hold
the stance in the present tense, edited when it changes, with a reason and a date
(https://loom.hypertheory-labs.dev/bedrock/venues/). What a venue lacks is the link
back, and phoropter has it: a plan whose decision rests on the stance anchors the
document that holds it, as a file (narrowed with `›`), and that file's page lists every
entry anchored there, the re-evaluation worklist, never maintained. Phoropter needs
nothing new and names no `.loom` path: where a team keeps its stances is its own
business. At most a guide sentence: "when a decision rests on a standing choice your
team records somewhere, anchor that document." `Holds while:` is one reason's
condition; an anchor to a stance is a condition many share.
