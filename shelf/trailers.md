# Seed: a `Plan:` trailer, and what reads it

Jeff and Claude, 2026-11-20. Commit history is supporting evidence; plans are the
primary way to read how code changed over time (`premises.md`). The link between them
is implied today (between a plan's added commit and its as-built's `head:`), which
breaks when two plans' work interleaves and vanishes in a squash.

- **The fact:** a git trailer, `Plan: plans/<file>.md`, on every commit that served a
  plan. Git parses trailers already (`git log --format='%(trailers:key=Plan,valueonly)'`,
  `git commit --trailer`). Precedents: the kernel's `Fixes:`, Gerrit's `Change-Id:`,
  which names an intended change across rewrites, as a plan's path does. This repository
  started writing them on 2026-11-20, so there's data before there's a reader.
- **The readers, which land with it** (one without the other is a distributed
  transaction):
  - blame with reasons: line → commit → plan, for every line the work touched, not only
    what a plan anchored (`shelf/file-pages.md`).
  - crossings: plan B's commits changed what plan A's Decision anchors, laid side by
    side with whether B revisits, cites or anchors A. The reader decides whether it was
    reconciled ("I'm not sure we really reconciled that"); phoropter prints the facts,
    not that word (`premises.md`). `git log -S` finds the commit that removed a quote.
  - work outside any plan: commits with no trailer, piling up in one area.
- **Squash, reframed:** people squash to make the log readable, to revert as a unit, to
  hide noise. Here the plan carries the story, so the log can be as noisy as the work
  was; revert-as-a-unit is "every commit naming this plan". Squashing becomes
  unnecessary, and harmless if it keeps one line. GitHub's squash likely buries other
  commits' trailers mid-message (unverified); the guide's squash section would say to
  keep the `Plan:` line.
- **Plans and merge conflicts:** plans are new files, so they rarely conflict as text.
  Two risks: two branches choosing the same plan name (the date usually differs; worth
  checking what phoropter does with a repeated name), and the real one, two branches'
  plans deciding opposite things at the same anchor. That's an unreconciled crossing,
  found after the merge.
- **Context:** Anthropic's Scion post (anthropic.com/research/redesigning-source-control,
  read as a summary): reviewers "examine the intent of the change", engineers' role "to
  tell Claude 'no'", blame computed as derived data. It says nothing about where intent
  or the no's are recorded. Those are plans and Declines.
