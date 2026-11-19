# Seed: documentation as a projection of the plans

Jeff, 2026-11-19 (marigold): scrolling the plans and as-builts, documentation looks like
a projection of them, the way the code is another. "A whole can of worms."

The evidence is that day's README drift: a hand-kept projection that drifted (a lens
example naming an entry that didn't exist, an install section older than `doctor`).

- **Jeff's version first, because it's buildable now:** plans "pinned" to a file. Any
  plan updating the README says why and which earlier plans brought it on. Then a new
  plan that touches the same code as one of those, or cites it, is a reason to check
  the README for drift. That's a walk over the graph phoropter already has (anchors,
  citations, revisits); the README's place page already lists every plan that anchored
  it. `phoropter affected --since` (`shelf/affected.md`) is most of it.
- **The finer version:** a sentence of documentation anchored to the entry that justifies
  it, so phoropter can say when a doc rests on a decision that's been revisited. Jeff's
  says which document to check; this says which sentence.
- **Plans are testimony, docs are present tense.** What's true now is close to
  computable: per thing decided, the newest Decision no revisit replaced.
- **Not the cart's event log:** it's deleted on conversion, on purpose. A projection
  reads plans and as-builts only.
