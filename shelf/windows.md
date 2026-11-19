# Seed: phoropter on Windows

From the persimmon cart (2026-11-19). The trial may run on Jeff's class VM, which is
Windows. `GOOS=windows go build` succeeds, but reading the code, three things probably
break there. None is tested; all are likely:
- **`phoropter url` never finds a server.** `servingURL` checks a server is alive with
  `p.Signal(syscall.Signal(0))`, which Windows doesn't support. Every server would look
  dead, and its record would be deleted.
- **The next free port doesn't happen.** `serve.go` moves on only when the error is
  `syscall.EADDRINUSE`. Windows reports `WSAEADDRINUSE`, so a second server would
  probably fail instead of taking 4174.

- **Indexing fails outright.** `Checkout` runs `sh -c 'git archive | tar -x'`. Windows
  has `tar.exe`, but `sh` exists only inside Git Bash, so from PowerShell every index of
  a commit fails. Git can write the archive to a file, or phoropter can untar it in Go,
  with no shell at all.

Others to look for when it's tried: the TUI's clipboard (`pbcopy`, OSC 52) and `open` commands, path separators in anchors, and
each indexer's toolchain on Windows. That last one is Jeff's real worry, and why
containers came up (`shelf/indexing.md`).
