# Seed: indexing, what's still open

Landed: indexers are programs, picked by an anchor's scheme, in `indexers/` or on `PATH`
as `phoropter-index-<scheme>` (seersucker); the index store, committed by `phoropter
index` and cached otherwise (batik); the guide for writing one (dimity); and
`phoropter doctor`, with indexers reporting their own `needs` (grosgrain). Still open:

- **Installable indexers.** A `go install`ed phoropter finds none (organdy's as-built).
  `indexers/go/`'s module is `phoropter-index-go`, so it can't be installed by name, and
  the TypeScript one isn't a Go program. Grosgrain declined fixing this until its own
  plan. Options: installable first-party indexers on `PATH`, or the Go indexer built
  into the binary as a fallback.
- **Containers, for indexers only** (Jeff): "if we use containers, it's for the
  indexers, not the phoropter binary", because getting every indexer working on every
  platform may be hard. The contract allows it already: a `phoropter-index-go` that runs
  scip-go in a container with the checkout mounted is just another indexer, and `doctor`
  could offer it beside the native install. The cost: an image to build and publish,
  Docker or Podman on the machine (admin rights, on the class's corporate machines), and
  old commits' toolchains, which make an image heavier, not lighter. Jeff, 2026-11-19 (tulle): "one more nickel toward 'or you can just docker run this
  thing'", since .NET and Java will each add a toolchain, and that's going to get weird.
- **Old commits.** Toolchains and module downloads for a commit years old are the hard
  part. The committed store answers for commits already indexed; nothing yet helps with
  one that isn't.
