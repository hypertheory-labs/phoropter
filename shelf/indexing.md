# Seed: where indexing lives, with a plugin for each language

From the TUI's brief and `tui/docs/next.md`. Phoropter indexes only with
scip-typescript, so a Go symbol anchor (the TUI's) doesn't resolve. SCIP is already the
plugin seam: the language-specific part is finding the package roots an anchor asks for
(`package.json`, `go.mod`, `pom.xml`, a `.csproj`) and running a command that writes
`index.scip`. The anchor's scheme (`scip-go gomod …`) names the indexer, so no
configuration. Indexers live on `PATH`; a missing one is a view status, not a build
failure; phoropter reads SCIP with its own bindings rather than scip-typescript's
`dist`; historical commits are the hard part (toolchains, module downloads), so cache
per commit and package. Part of a larger idea: a `phoropter` CLI, `gh`- or
`kubectl`-style, with UIs and indexers as plugins.

The store this needs, where indexes live and how they're kept, is decided in batik (the
`phoropter` command).

**From organdy's as-built:** a `go install`ed phoropter finds no first-party indexers.
They're programs in `indexers/`, found only from a clone (or `PHOROPTER_INDEXERS`), and
`indexers/go/` can't be `go install`ed by name. The shipping plan needs one of:
installable first-party indexers (`phoropter-index-go`, `phoropter-index-typescript` on
`PATH`), or the Go indexer built into the binary as a fallback.

**From grosgrain (2026-11-19):** `phoropter doctor` now says plainly that a binary outside
a clone has no first-party indexers, and how to get them, and grosgrain declined fixing
that until its own plan. That plan is probably next after the server.

**Containers, for indexers only (Jeff, persimmon cart):** "if we use containers, it's for
the indexers, not the phoropter binary", because getting every indexer working on every
platform may be hard. The contract already allows it: a `phoropter-index-go` that runs
scip-go in a container with the checkout mounted is just another indexer on `PATH`, and
`doctor` could offer it beside the native install. The cost: an image to build and
publish, Docker or Podman on the machine (admin rights, on the class's corporate
machines), and old commits' toolchains, which make an image heavier, not lighter.
