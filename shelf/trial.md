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

Later that night (Jeff cloned it to `../files-extension`): the agent wrote calico (how
we work), osier (the pipeline test), and marram, the extension's first real plan, with
four genuine Declines, Jeff's words in the reasons, attribution that cuts against him
("this surprised him"), a plan-entry anchor to osier, and an honest as-built. It found
the design itself: a reordered folder is a lens whose paths share a directory. Its
indexes, made on Windows and committed, built the site on a Mac in half a second with
no indexer run. Osier's one `no-symbol` is correct drift. Two more for phoropter: a
`where (new):` glob stopped the whole site on Windows (dupioni fixed it), and a failed
rebuild was visible only on the server's stderr (the next plan). The agent rewrote its
unpushed history to fix the anchor, reasoning from the rule's purpose ("nobody's
anchors move underneath them, because there was nobody").

Still open:
- **The pasted pointer kept `init`'s indentation**, so in AGENTS.md its paragraph is a
  Markdown code block. `init` could print the pointer flush left, between rules.
- **A stale install.** `go install …@latest` right after a push gave a binary from before
  it (Go's proxy lags), and nothing told the newcomer. `GOPROXY=direct` fixed it. Maybe
  `doctor` says its own version, or compares it with the latest.
- **Not observed:** a second server taking the next port on Windows, cleanly.
- **The extension** stays in limbo there; Jeff won't build it on Windows if he can help it.
  Whoever picks it up gets a repository already working this way.
- **A second clean environment:** DevPod (Jeff has it) for a throwaway Linux machine,
  if a later trial wants one.
