# phoropter

Working name: the optometrist's "one, or two?" device. A tool that lets a person and
an agent read code through the plans that shaped it. The thinking happens in
`~/Projects/aide`; this repo is the product.

- **Plans:** `plans/YYYY-MM-DD-<cart>.md`, committed before the work and never edited
  after. What changed during the work goes in `<plan>.as-built.md` beside it, as
  `## Changed:` entries; departing from the plan is allowed, saying so is the point.
  The format is defined in `~/Projects/aide/plans/2026-10-30-lanyard.md`.
- **The first plan is also the brief:** `plans/2026-10-30-gimlet.md`. Its two Guesses
  are what the as-built should answer.
- **Test data:** `~/Projects/mock-control`, whose plan is `plans/2026-10-29-pumice.md`.
- **Carts:** one human and one agent, gitignored in `.loom/cart/`. The agreement is
  `~/Projects/aide/.loom/externals/git.hypertheory-labs.dev/loom/cart/cart.md`. A cart
  produces one plan; if it wants two, it splits. A `shelf/` with no `current/` is a seed
  waiting for a cart to open.
- **Claude commits; Jeff pushes.**
- **Running:** `npm run dev` serves mock-control at `90c65b9` (expects `../mock-control`;
  the tests also read `../aide`, and skip what isn't there);
  `node src/build.ts <repo> [commit]` writes `site/`, which is gitignored. `npm test`,
  `npm run check`. Node 24+ runs the `.ts` sources directly; there's no build step.
