# Seed: UIs as plugins, and what `phoropter` opens

Organdy shipped the TUI inside the binary as `phoropter ui`, and nainsook gave it `y`,
`Y` and `b`. The server questions that started here (closing the UI stops the server;
`ui` starts one when there's none) are in `shelf/one-server.md` now. Still open:

- **Other UIs as plugins.** A UI is a reader of the served pages, never of phoropter's
  internals, found on `PATH` as `phoropter-<name>` like `gh` extensions. The TUI being
  inside the binary means the plugin contract hasn't been tested by anyone else's UI yet.
- **A UI as the default** when you run `phoropter` (Jeff), "with the address displayed
  somewhere so it can be handed to the agent". A way to run the server alone stays, for
  an agent working while the UI is closed.
- **A short name.** "Phoropter" will be misspelled and aliased like `kubectl` (Jeff). A
  short command installed beside it is the shipping plan's question.
