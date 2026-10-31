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
- **Drafts:** `drafts/plan.draft.md`, one at a time, as the guide says (`drafts/` is
  gitignored here; we work on main). Put one up when you're unsure or the understanding looks thin, or when Jeff asks to see every
  plan for a session. Never an approval gate; silence means proceed. Most work just
  goes. Serve this repo (`node src/serve.ts . HEAD`) and it's at `draft/`; Jeff
  annotates in nvim. Convert when confident, and delete the draft and its notes.
- **Notes on landed plans: check `drafts/` at the start of a session.** A note on a
  landed plan (`drafts/<plan's file name>.notes.md`) says the plan matters to something
  now. Raise any you find before other work, and answer in kind: a revisit if its ground
  moved, a citation in the current draft if it bears on it, or a note in reply. Never
  edit a landed plan.
- **Carts are thinking, not drafting.** `.loom/cart/` (gitignored) is where Jeff and
  Claude think something through; a cart's result is "let's draft a plan from this."
  Phoropter knows nothing about carts. A `shelf/` holds seeds for later (many from
  the TUI's old brief); restate a seed into a draft when it's picked up, and delete it
  then.
- **`.loom/` is lineage, not a dependency.** Copies of documents this work descends
  from (the annotating and cart agreements). Read them, but don't reference `.loom`
  paths from code or docs, and don't require loom tooling. What's adopted is restated in
  this repository's own words.
- **When you refer to a plan in prose, say what it's about**, not just its name. A
  sentence made of plan names reads as Dr. Seuss.
- **Test data:** `~/Projects/mock-control` (its own plans in `plans/`), and aide's
  `plans/`, which are mostly plan-to-plan anchors.
- **The rels and microdata are a public API.** The profile defines them, and readers
  other than a browser depend on them. Renaming one breaks someone; it's a plan, not a
  tidy-up.
- **`tui/` is a reader of that API, first-party** (dowlas). It's a Go terminal reader that
  reads only served or built pages, never `src/`, and keeps a ledger of every place it
  had to read past the profile. Its gaps are phoropter's to close. A session working
  in `tui/` reads `tui/CLAUDE.md`. Its plans are in `plans/` with everyone's.
- **Claude commits; Jeff pushes.** This file is Claude's: Jeff doesn't edit it. Keep it
  short and true for the next session.
- **Running:** `npm run dev` serves mock-control at `90c65b9` on localhost (expects
  `../mock-control`; pass a later commit to see the newer plans); the tests also read
  `../aide`, and skip what isn't there. `node src/build.ts <repo> [commit]` writes
  `site/`, which is gitignored. `npm test`, `npm run check`. Node 24+ runs the `.ts`
  sources directly; there's no build step.
