# Seed: tests grounded in plans

Jeff, 2026-11-20: tests can stop carrying the burden of explaining business rules,
requirements and technical constraints, because a plan grounds them (`premises.md`:
plans are testimony, tests are present tense). Is the plan vocabulary enough for that
grounding?

What exists:
- **Plan to test:** any entry can anchor a test file or test function. Several as-builts
  do (sorrel's `Chose:` pinned a lenient parse "by a test"). So a test file's page
  already lists the entries that anchored it.
- **Test to plan:** only by convention. This repository's comments name plans
  ("(gingham)"), and nothing resolves those.
- **Answered:** the guide gives it a verdict and "how much evidence that rests on", in
  prose. The parser reads anchors on any entry, so an `Answered:` with a `where:` at the
  test that answered it may already work. Untested, and the guide doesn't mention it.

What's open:
- **An Answered entry anchors its evidence.** When a test answers a guess, the as-built
  points at it. Then the test's page says "this answers that guess", and its reason is
  one click away. Possibly only a guide sentence.
- **A guess says a test could watch it.** Some guesses are about code ("no anchor text
  can stop a build"), and some are about people ("drafts get read"). A code guess could
  name the test that would watch its condition, maybe `where (new):` at a test. Careful:
  a guess that a test enforces is close to a Decision. Maybe the as-built is where it
  turns into one ("held, and now watched by…").
- **Reasons that drift.** When a test's anchoring entry is revisited, the test's reason
  moved. That's `affected` (`shelf/affected.md`) asking about tests.
