# phoropter-tui

A terminal reader for a phoropter site, the plans-and-anchors substrate that phoropter
(this repository's root) generates from a repo at one commit. It's an experiment in two
ways: a tool for a human (and, through the same pages, an agent) asking "what were they
thinking when they wrote this?", and a second reader of phoropter's HTML that finds
where its API is thin. It came into this repository with its history (dowlas: the TUI
moves in, as `tui/`). This file is Claude's; Jeff doesn't edit it. Keep it short and
true.

- **The boundary: the TUI reads only what phoropter serves.** It's tested against
  served or built pages. It never reads `src/`, and never imports or shells out to
  phoropter's code beyond running its dev server. The public API is the boundary, not
  the directory. If the ledger goes quiet, it must be because phoropter closed a gap.
- **Read phoropter's `guide.md` and the site's `profile/` first.** Those are the
  contract, not `src/`.
- **Reasons come from the substrate. The TUI computes only local facts** (file tree,
  whether a local file matches what a page says, working-tree drift). If the TUI
  inferred reasoning the HTML lacks, the person would see more than the agent. Gimlet
  declined a separate client app for this reason; the TUI is the "projection of the same
  pages" it allowed. No git history in the TUI.
- **It writes only notes, in `drafts/`,** by `docs/annotating.md` (`internal/notes`): a
  draft's, and requests to revisit landed plans. Never the draft, never a landed plan
  (nankeen declined editing plans), never the substrate. It reads notes back from the
  dev server's pages, never from the files.
- **The gaps ledger is half the point.** Whenever the code reads past the profile, it
  calls `Ledger.Note` with an id from the catalog in `internal/substrate/gaps.go`. Don't
  work around a gap silently. `go run . gaps <site>` crawls and reports. When phoropter
  closes a gap, delete the workaround and the catalog entry, and add it to a test that
  keeps it closed. When phoropter declines one, keep it and say which plan declined it.
- **Asking phoropter for something is a draft now,** in this repository, after checking
  phoropter's declines. The old brief (`docs/for-phoropter.md`) is superseded by seeds
  in `shelf/`.
- **Plans** are in the root `plans/`, by the guide, with phoropter's. A TUI plan can
  anchor and revisit phoropter entries now. Use file anchors with a quote (paths from
  the repository root: `tui/internal/tui/screens.go`). Phoropter indexes only
  TypeScript, so a Go symbol anchor won't resolve yet. The TUI's plans from before the
  move anchor `internal/…` and show `no-file` at later commits; they're read as written.
- **Drafts:** one, `drafts/plan.draft.md`, and requests to revisit landed plans beside it
  in `drafts/` (tarlatan). Serve the repository with
  `node src/serve.ts . HEAD` from the root and point the TUI at it. To try the TUI on a
  draft, `mkdir -p drafts && cp tui/testdata/sample.draft.md drafts/plan.draft.md`.
- **Working with Jeff.** Jeff reads drafts before code, so propose the bigger or bolder
  version and let his notes trim it. An agent never gets a veto: a draft weighing an
  idea Claude thinks is bad says so up front, plans it properly anyway, and defaults its
  prompt to Jeff's idea. Keep the ceremony in proportion: small fixes go straight to
  `plans/`, or need no plan. Stage commits by explicit path, never `git add -A`: Jeff
  drops files in that aren't for the commit. Gate commits on `go test ./...` itself,
  not piped through anything.
- **When you refer to a plan in prose, say what it's about.**
- **Where this is going:** `docs/next.md` (file mode, what's already at a draft's
  places, `init`, `why`, reasons to re-check, and a phoropter CLI with UIs and indexers
  as plugins, `gh`- or `kubectl`-style, where this TUI is one UI).
- **Layout:** `internal/microdata` (generic WHATWG microdata), `internal/substrate`
  (typed pages, the ledger, the crawl), `internal/notes` (the notes file; the only
  thing that writes), `internal/tui` (Bubble Tea v1). Page kind is decided from the
  main item's properties, not the URL.
- **Running** (from `tui/`): `go build -o bin/phoropter-tui .`, then
  `bin/phoropter-tui [url|dir]` (default `http://localhost:4173/`), and `go test ./...`.
  `npm test` at the root doesn't run these, and that's deliberate. The tests read
  `testdata/`, sites built by `node src/build.ts <repo> <commit> --out …` at the root.
  Rebuild them in the same work as a profile change.
- **Dependencies are pinned** for Go 1.24: bubbletea v1.3.4, lipgloss v1.1.0,
  `charmbracelet/x/ansi` v0.8.0. `go mod tidy` will bump `x/ansi`, which breaks
  lipgloss's `cellbuf`. Pin it back with `GOTOOLCHAIN=local go get …@v0.8.0`.
- **Claude commits; Jeff pushes.**
