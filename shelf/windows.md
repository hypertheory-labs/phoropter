# Seed: phoropter on Windows

From the persimmon cart (2026-11-19). The trial may run on Jeff's class VM, which is
Windows. `GOOS=windows go build` succeeds, but reading the code, two things probably
break there. Neither is tested; both are likely:
- **`phoropter url` never finds a server.** `servingURL` checks a server is alive with
  `p.Signal(syscall.Signal(0))`, which Windows doesn't support. Every server would look
  dead, and its record would be deleted.
- **The next free port doesn't happen.** `serve.go` moves on only when the error is
  `syscall.EADDRINUSE`. Windows reports `WSAEADDRINUSE`, so a second server would
  probably fail instead of taking 4174.

Others to look for when it's tried: `Checkout` runs `sh -c 'git archive | tar -x'`, the
TUI's clipboard (`pbcopy`, OSC 52) and `open` commands, path separators in anchors, and
each indexer's toolchain on Windows. That last one is Jeff's real worry, and why
containers came up (`shelf/indexing.md`).
