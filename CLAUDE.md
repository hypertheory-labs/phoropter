# phoropter-tui

A terminal reader for a phoropter site, the plans-and-anchors substrate that
`../phoropter` generates from a repo at one commit. It's an experiment in two ways: a
tool for a human (and, through the same pages, an agent) asking "what were they thinking
when they wrote this?", and a second reader of phoropter's HTML that finds where its API
is thin. This file is Claude's; Jeff doesn't edit it. Keep it short and true.

- **Read phoropter's `CLAUDE.md` and `guide.md` first.** The guide is the plan format;
  the site's `profile/` page defines every rel and microdata property. Those, not
  phoropter's source, are the contract.
- **Reasons come from the substrate. The TUI computes only local facts** (file tree,
  blob match, working-tree drift). If the TUI inferred reasoning the HTML lacks, the
  person would see more than the agent. Phoropter's gimlet plan declined a separate
  client app for this reason; this TUI is the "projection of the same pages" it allowed.
  No git history in the TUI.
- **The gaps ledger is half the point.** Whenever the code reads past the profile
  (prose, unprofiled markup, nothing to show), it calls `Ledger.Note` with an id from
  the catalog in `internal/substrate/gaps.go`. Don't work around a gap silently; note it.
  `phoropter-tui gaps <site>` crawls and reports. When phoropter closes a gap, delete the
  workaround and the catalog entry. When phoropter *declines* one, keep it and say which
  plan declined it and why (see `answer-elsewhere`).
- **Check phoropter's declines before asking for something.** It records what it turned
  down and why; several things a reader wants were declined, some for sequencing that
  has since been met, some on principle. Asks go to phoropter as briefs
  (`docs/for-phoropter.md`); never edit `../phoropter` or `../mock-control`.
- **Where this is going:** `docs/next.md` (file mode, cart mode, `why`, reasons to
  re-check). The cart (`/cart`, proto-plans the pair drafts) is phoropter's, arriving
  after 2026-10-30. The entry point is a plan for now; a file view may come, or an LSP
  adapter instead.
- **This repo uses the plan format itself** (since 2026-10-31): plans in `plans/`, by
  phoropter's guide, and a draft as `plan.draft.md` at the root (gitignored; we work on
  main). Serve it with `node src/serve.ts ../phoropter-tui --port 4174` from
  `../phoropter`. To try the TUI on a draft, `cp testdata/sample.draft.md plan.draft.md`. Use file anchors with a quote. Phoropter indexes only TypeScript, so a
  Go symbol anchor won't resolve yet. Don't backfill plans for work already done; its
  reasons are here and in the commits.
- **`.loom/` is lineage, not a dependency.** Jeff keeps copies of documents this work
  descends from there (the annotating agreement among them). Read them, but don't
  reference `.loom` paths from code or docs, and don't require loom tooling. What we
  adopt is restated here in our own words, as phoropter does.
- **When you refer to a plan in prose, say what it's about,** as phoropter's file asks.
- **Layout:** `internal/microdata` (generic WHATWG microdata, nothing phoropter-specific),
  `internal/substrate` (typed pages, the ledger, the crawl), `internal/tui` (Bubble Tea
  v1). Page kind is decided from the main item's properties, not the URL.
- **Running:** `go build -o bin/phoropter-tui .`; `bin/phoropter-tui [url|dir]` (default
  `http://localhost:4173/`, phoropter's `npm run dev`); `go test ./...`. The tests read
  `testdata/`, sites built by `node src/build.ts <repo> <commit> --out …` in
  `../phoropter`; rebuild them when the profile changes.
- **Dependencies are pinned** for Go 1.24: bubbletea v1.3.4, lipgloss v1.1.0,
  `charmbracelet/x/ansi` v0.8.0. `go mod tidy` will bump `x/ansi`, which breaks
  lipgloss's `cellbuf`; pin it back with `GOTOOLCHAIN=local go get …@v0.8.0`.
- **Commit when Jeff asks; Jeff pushes.**
