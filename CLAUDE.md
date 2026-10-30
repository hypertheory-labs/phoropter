# phoropter

Working name: the optometrist's "one, or two?" device. A tool that lets a person and
an agent read code through the plans that shaped it. The thinking happens in
`~/Projects/aide`; this repo is the product. The point is Jeff's understanding keeping
pace with the work, not coverage.

- **Plans:** `plans/YYYY-MM-DD-<name>.md`, committed before the work and never edited
  after. Phoropter owns the format, and `guide.md` states it; every site serves it at
  `guide/`. Read it before writing a plan or changing how phoropter reads them. A
  format change is a phoropter plan, and the guide changes in the same work. Aide's
  format plans (lanyard, grommet, bodkin, awl) are history, not the rules.
- **Changing your mind about an earlier plan is a revisit**: its own plan, with
  `revisits:`, asked in that plan's frame, saying what moved.
- **Drafts:** when you're unsure or the shared understanding looks thin, put the plan up
  as a draft in the cart before committing it, so Jeff can question it before it's
  testimony. Never an approval gate: every open question has a fallback, and silence
  means proceed. Most work just goes.
- **Carts:** one human and one agent, gitignored in `.loom/cart/`. The agreement is
  `~/Projects/aide/.loom/externals/git.hypertheory-labs.dev/loom/cart/cart.md`; carts
  here act it out. A cart produces one plan; if it wants two, it splits. A `shelf/` with
  no `current/` is a seed waiting for a cart to open.
- **When you refer to a plan in prose, say what it's about**, not just its name. A
  sentence made of plan names reads as Dr. Seuss.
- **Test data:** `~/Projects/mock-control` (its own plans in `plans/`), and aide's
  `plans/`, which are mostly plan-to-plan anchors.
- **The rels and microdata are a public API.** The profile defines them, and readers
  other than a browser may depend on them (Jeff may build a TUI on them, in another
  repo). Renaming one breaks someone; it's a plan, not a tidy-up.
- **Claude commits; Jeff pushes.** This file is Claude's: Jeff doesn't edit it. Keep it
  short and true for the next session.
- **Running:** `npm run dev` serves mock-control at `90c65b9` on localhost (expects
  `../mock-control`; pass a later commit to see the newer plans); the tests also read
  `../aide`, and skip what isn't there. `node src/build.ts <repo> [commit]` writes
  `site/`, which is gitignored. `npm test`, `npm run check`. Node 24+ runs the `.ts`
  sources directly; there's no build step.
