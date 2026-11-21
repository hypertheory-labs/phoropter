# Seed: indexing, what's still open

Landed: indexers are programs, picked by an anchor's scheme, in `indexers/` or on `PATH`
as `phoropter-index-<scheme>` (seersucker); the index store, committed by `phoropter
index` and cached otherwise (batik); the guide for writing one (dimity); and
`phoropter doctor`, with indexers reporting their own `needs` (grosgrain). Still open:

- **Installable indexers.** Tulle made the Go one installable (`go install
  …/cmd/phoropter-index-scip-go@latest`). The TypeScript one still needs a clone: an npm
  package is the likely shape, once the trial says a clone is the friction.
- **Containers, for indexers only** (Jeff): "if we use containers, it's for the
  indexers, not the phoropter binary", because getting every indexer working on every
  platform may be hard. The contract allows it already: a `phoropter-index-go` that runs
  scip-go in a container with the checkout mounted is just another indexer, and `doctor`
  could offer it beside the native install. The cost: an image to build and publish,
  Docker or Podman on the machine (admin rights, on the class's corporate machines), and
  old commits' toolchains, which make an image heavier, not lighter. Jeff, 2026-11-19 (tulle): "one more nickel toward 'or you can just docker run this
  thing'", since .NET and Java will each add a toolchain, and that's going to get weird. Jeff, 2026-11-21, after asking about C++ for an
  ESP32 repository: "much more appealing". scip-clang (Sourcegraph, beta) has binaries
  for x86_64 Linux and macOS only, Windows "being explored", so a container is the
  only way it runs on Windows. It also needs a `compile_commands.json`, so for an old
  commit the container has to configure the build before indexing, which is the "old
  commits" problem below in its sharpest form. And its Linux binary is x86_64, so on
  Apple Silicon or a Jetson the container runs emulated.
- **What a throwaway draft found** (2026-11-21, worktree `../phoropter-containers`,
  branch `explore/indexer-containers`, never pushed; remove both when done with it):
  - Containers are nearly free for phoropter: a shim on `PATH` that runs `docker run`
    meets the contract, `--describe` reports Docker as a `need`, and the only change in
    phoropter is `doctor` printing an "or, with Docker" line. One image per indexer,
    built by a workflow on a tag (GitHub's registry, or Jeff's Gitea for now).
  - C++ is the hard part, and a container doesn't make it easier: scip-clang needs a
    configured build. Phoropter keeps only definitions, so a parser-only C++ indexer
    (tree-sitter), spelling symbols as scip-clang does, works on every commit and
    platform. Its cost is `#ifdef`. Jeff chose it.
  - First steps if it happens: a spike of tree-sitter's Go binding on Windows (cgo
    needs a C compiler there, the friction all this is about), then a fresh agent
    builds the indexer from `indexers/README.md` alone and becomes the reference
    (Jeff's idea; dimity's experiment again).
  - Correction: committed indexes cover a plan's own commits, but HEAD's index is in
    the local cache, so a reader without the indexer loses the "now" view and
    `affected` misses symbol anchors.
- **Old commits.** Toolchains and module downloads for a commit years old are the hard
  part. The committed store answers for commits already indexed; nothing yet helps with
  one that isn't.
