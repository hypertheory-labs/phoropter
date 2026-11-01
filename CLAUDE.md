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
- **A plan lands whole.** If it would leave a gap that only a later plan fills, reorder
  them or merge them. Two plans that only work together are a distributed transaction
  (Jeff); the index store was shelved until the `phoropter` command's plan for that.
- **Changing your mind about an earlier plan is a revisit**: its own plan, with
  `revisits:`, asked in that plan's frame, saying what moved.
- **Drafts:** `drafts/plan.draft.md`, one at a time, as the guide says (`drafts/` is
  gitignored here; we work on main). Put one up when you're unsure or the understanding looks thin, or when Jeff asks to see every
  plan for a session. Never an approval gate; silence means proceed. Most work just
  goes. Run `phoropter` (`npx phoropter`, from anywhere in the repo) and it's at `draft/`; Jeff
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
- **phoropter is Go, one binary** (organdy): `main.go` is the command, `internal/site/` is
  phoropter itself (plans, resolution, the store, indexers, rendering, the dev server),
  and the TUI ships in it as `phoropter ui`. The TypeScript is gone; only the TypeScript
  indexer is Node, because scip-typescript is. `testdata/golden/` holds sites the Node
  version built, and the Go version must match them in microdata.
- **`tui/` is a reader of that API, first-party** (dowlas). It reads only served or built
  pages, never `internal/site/`, and a test in `tui/app/` fails the build if any TUI
  package imports phoropter's own. It keeps a ledger of every place it had to read past
  the profile. Its gaps are phoropter's to close. A session working
  in `tui/` reads `tui/CLAUDE.md`. Its plans are in `plans/` with everyone's.
- **Claude commits; Jeff pushes.** This file is Claude's: Jeff doesn't edit it. Keep it
  short and true for the next session.
- **The command** (batik): `phoropter` in any repository serves it (port 4173, or the
  next free one; `phoropter url` prints where), `phoropter build` writes the static
  site, `phoropter index` commits the indexes plans need into `.phoropter/index/`, and
  `phoropter ui` opens the TUI. Served at HEAD, it follows HEAD; at a terminal it has a
  key menu (`h` + enter), and in the background it reads nothing (poplin).
  `phoropter guide` prints the guide with no server, and `phoropter indexers` lists what
  this machine can index and where each was found. Committing indexes is the one write into a repository,
  and only when run on purpose. Other indexes are cached in `~/.cache/phoropter/`.
  `go install github.com/hypertheory-labs/phoropter@latest` installs it; here,
  `go build -o bin/phoropter .`, and Jeff's `~/.local/bin/phoropter` is a symlink to
  `bin/phoropter`. A binary built in this repository finds `indexers/` by walking up
  from itself; anywhere else, indexers come from `PATH`.
- **Indexers** (seersucker) are programs in `indexers/` (TypeScript, Go), or on `PATH` as
  `phoropter-index-<scheme>`; a symbol's first word picks one. `indexers/README.md` is
  the contract, written so an agent can build one from it alone. `phoropter symbols
  <file>` lists symbols to anchor (and says why when there are none), and `phoropter index --check <dir>` tests an
  indexer. The Go one needs Go 1.25+ and scip-go, which moved to
  `github.com/scip-code/scip-go` (`go install …/cmd/scip-go@latest`); it reads 0.1 and
  0.2 alike.
- **Running:** `go test ./...` runs everything, the golden comparison included (the
  tests read `../mock-control`, and skip what needs it when it isn't there).
  `phoropter ../mock-control --at 90c65b9` serves mock-control's older plans. `npm
  install` is only for the TypeScript indexer. The TUI's dependency pins (Bubble Tea
  v1.3.4, lipgloss v1.1.0, `charmbracelet/x/ansi` v0.8.0) still hold: `go mod tidy` can
  bump `x/ansi` and break lipgloss's `cellbuf`, so pin it back if it does.
