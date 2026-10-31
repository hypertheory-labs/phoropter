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
