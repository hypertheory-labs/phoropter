# Seed: contributing to a project that uses plans

From Jeff, thinking about phoropter as an open-source project: "Want to contribute? File
an issue with a phoropter plan first! And the as-built would be such a great thing to
attach to the PR."

The shape it suggests:
- **The issue carries the plan,** or a draft of it, before any code. Maintainers
  annotate it the way Jeff annotates a draft, with notes and answers to its prompts, and
  the contributor revises. Nobody reviews code that solves the wrong problem.
- **The pull request carries the plan and its as-built,** so the review reads what was
  decided, what departed, and what the guesses turned out to be, before the diff.
- **A maintainer's decline is a recorded decline,** findable later, not a closed issue
  nobody reads.

Questions for when it's picked up: whether `phoropter` reads a plan from an issue at all
or only from a branch, how a contributor's draft and a maintainer's notes travel (a
branch with `drafts/` committed, as tarlatan describes), and whether the guide says
anything or a CONTRIBUTING file does.
