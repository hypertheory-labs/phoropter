# Seed: UIs as plugins, and what `phoropter` opens

The second plan toward the `phoropter` CLI, after the one that makes the command. From
the conversation about it, and Jeff's notes on the first plan's draft:

- **A UI is a reader of the served pages,** chosen by the person, found the way the
  plugin contract says (on `PATH` as `phoropter-<name>`, like `gh` extensions, or
  registered, like `kubectl` plugins). It never reads phoropter's internals. The TUI in
  `tui/` is the first one, and installing it like anyone else's is the honest test of the
  contract.
- **Jeff thinks a UI is the right default** when `phoropter` serves, "with the address
  displayed somewhere so it can be handed to the agent". `phoropter url` (from the first
  plan) and the UI's own header can both show it.
- **Jeff: closing the UI could stop the server too.** Simple and honest about what's
  running. The caution: an agent reading the pages loses them mid-task when the person
  quits the UI. So keep a way to run the server alone (`phoropter serve`), for the agent
  working while the UI is closed.
- **The address in the TUI's footer** (Jeff), with a key that opens it in a browser.
- **Indexers are plugins too, but a different contract:** a step in the build, chosen by
  the scheme each anchor names (`scip-go gomod …`), not by the person. See
  `shelf/indexing.md`. They may share one way of being found.
- **A short name.** "Phoropter" will be misspelled and aliased like `kubectl` (Jeff). A
  short command installed beside it is the shipping plan's question.
- **`phoropter ui` starts the server when there's none (Jeff, 2026-11-19, on muslin's
  template).** If a server is running, it reports the address; if not, it starts one and
  reports that. Symmetry with the template's "the agent runs the server": either of us
  starts it, whoever gets there first. For the server and TUI friction cart, with the
  5.3s first request after a commit.
