# Seed: the trial, and what's left of it

The first repository outside phoropter, set up fresh: Jeff's Windows VM, 2026-11-19, a
VS Code extension (ordered files, staging, without touching the file system), a fresh
agent given only `init`'s pointer. The agent was Copilot in VS Code (it picked Claude
Fable 5.5), reading AGENTS.md. It converted two plans: "How we work here", and a
pipeline test that anchored a TypeScript symbol. What it found became plans: the
TypeScript indexer's paths and the `symbols` → `doctor` loop, the guide's and pointer's
wording, and `init`'s template on an empty repository.

What held: `init`'s refusals and walkthrough, `doctor`, the TUI in VS Code's terminal,
`phoropter url` and indexing on Windows after chiffon, Node and scip-typescript on
Windows once paths were fixed. The conversions were faithful: answers folded and
attributed, an overruled Decline recorded honestly ("Copilot, then Jeff"), and the
agent left its fix to phoropter's clone uncommitted, "that repository has its own
plans". Two plans in, Jeff: it "seems much more entrained with the whole thing now".
And told about ordered files, it said "oh, you mean sort of like lenses in phoropter":
the concept reads to an outsider; whether a user needs one is the extension's question.

Still open:
- **A stale install.** `go install …@latest` right after a push gave a binary from before
  it (Go's proxy lags), and nothing told the newcomer. `GOPROXY=direct` fixed it. Maybe
  `doctor` says its own version, or compares it with the latest.
- **Not observed:** a second server taking the next port on Windows, cleanly.
- **The extension** stays in limbo there; Jeff won't build it on Windows if he can help it.
  Whoever picks it up gets a repository already working this way.
- **A second clean environment:** DevPod (Jeff has it) for a throwaway Linux machine,
  if a later trial wants one.
