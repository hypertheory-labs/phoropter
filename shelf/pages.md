# Seed: publish the site, on GitHub Pages or anywhere static

From Jeff: "the ability to publish the site as a github pages thing." Gimlet chose a
static site generator so a site could be "published, compared, or read from `file://`",
and batik's index store makes it cheap to do in CI.

- **A workflow** on push to main: check out with full history (plans name old commits),
  set up Node and the indexers' toolchains, `phoropter build`, upload to Pages.
- **The committed index store pays off here:** the runner never re-indexes a plan's old
  commit, which may need a toolchain the runner doesn't have. Only now gets indexed.
- **A built site holds only testimony.** No drafts, no notes on landed plans. Work in
  progress stays on the dev server.
- **The agent gets a stable public URL** for the same hypermedia API.

To weigh: some repositories' plans shouldn't be public (opt in, never default); Pages
serves under `/<repo>/`, and phoropter's links are relative, which should already work
but needs checking; and whether it's a `phoropter publish` command, a workflow file
that `init` writes, or just documentation.
