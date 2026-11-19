---
base: ee37fc3
---

# `phoropter doctor`: what this machine is missing, said plainly

On a new machine on 2026-11-19, `phoropter symbols` on any Go file said only `exit
status 1`. The Go indexer had said exactly what was wrong ("scip-go isn't installed: go
install github.com/scip-code/scip-go/cmd/scip-go@latest"), but it runs under `go run`,
which adds `exit status 1` below its last line. Phoropter keeps only the last line of
an indexer's error. So the honest message was there, and phoropter cut it off.

That's the small half of this plan. The larger half is a newcomer's first hour.
Phoropter is one binary, but anchoring symbols needs an indexer per language, and each
indexer needs its language's tools. Nothing says what's missing until something fails.
This is the friction that stands between a fresh machine and the trial (a new repository,
`phoropter init`, a fresh agent). `doctor` says, for the repository you're in, what
works, what doesn't, and the command that would fix each thing. It runs none of them.

## Decision: An indexer's error shows its own last line, not `go run`'s
- where: ``scip-go gomod github.com/hypertheory-labs/phoropter . `github.com/hypertheory-labs/phoropter/internal/site`/RunIndexer().``
- quote: `lines := strings.Split(msg, "\n")`

**Because (measured):** reproduced with a program that prints a message and exits 1
under `go run`: stderr is the message, then `exit status 1`. `RunIndexer` drops a
trailing `exit status N` line before taking the last one. Any indexer run through
`go run` gets its own words back.

## Decision: `phoropter doctor` reports, and runs nothing
- where: `main.go`
- quote: `case "init", "build", "index", "url", "symbols", "ui", "indexers", "guide", "help", "-h", "--help":`

**Because (Jeff and Claude):** in the repository you're in, it reports:
- whether git is there,
- whether phoropter's own indexers were found, and if not, why (a `go install`ed binary
  finds none; see the Decline below),
- each language this repository's plans anchor symbols in, by the scheme in each
  anchor, and whether an indexer for it is here and ready,
- for anything that isn't, the command that would fix it.

It doesn't install anything. Piped install scripts are what everyone does, and Jeff
hates running them (he hands them to Claude: "look kosher?"). Something that tells you
the command and leaves the running to you, or to your agent with you watching, is the
same help without that worry. The agent can read `doctor`'s output as easily as a
person can. Offering to run a fix on a yes was asked, and Jeff said no.

## Decision: An indexer says what it's missing, in `--describe`
- where: ``indexers/README.md › Writing an indexer for phoropter › The contract › `<program> --describe` ``
- quote: ``- **`extensions`:** the file extensions it indexes. `phoropter symbols <file>` picks``

**Because (Claude, not asked):** only the indexer knows what it needs: scip-go for Go,
`node_modules` for TypeScript, an SDK for .NET. If phoropter knew that itself, every new
indexer would need a phoropter change, and other people's indexers (Jeff's roslyn one)
couldn't take part. So `--describe` gains an optional `missing`: a list of what it
couldn't find, each with what to run. Leaving it out means "nothing missing, as far as I
know", so the roslyn indexer keeps working unchanged. `doctor` shows it, and
`phoropter index --check` checks its shape when it's there. The contract is public, so
`indexers/README.md` changes in this plan.

## Decision: Phoropter's two indexers report what they're missing
- where: `indexers/go/main.go`
- quote: `emit(map[string]any{"indexer": "phoropter-index-go", "version": version, "scheme": "scip-go", "extensions": []string{".go"}})`

**Because (Claude, not asked):** the Go indexer already knows how to look for scip-go,
in `findScipGo`. `--describe` runs the same check and reports it. The TypeScript
indexer checks for scip-typescript in its `node_modules`, which `npm install` in a clone
provides. Both say the command, as the Go indexer's error already does.

## Decision: When `symbols` fails because something's missing, it points at `doctor`
- where: `main.go`
- quote: `func symbols(repo *site.Repo, root, target, at, scheme string) {`

**Because (Claude, not asked):** `symbols` is where a newcomer first finds out. The
indexer's own message comes first. Then one line: `phoropter doctor` lists everything
this repository needs.

## Decline: Making the indexers installable with `go install`
**Because (Claude, not asked):** a `go install`ed phoropter finds no indexers (organdy's
as-built), and `indexers/go/`'s module is named `phoropter-index-go`, so it can't be
installed by name. Fixing that means moving the indexers somewhere installable, under
names like `phoropter-index-go`, and the TypeScript one isn't a Go program at all. It's
a plan of its own, and probably the next one: `doctor` makes the gap visible, and the
trial on a clean machine will hit it first. For now, `doctor` says plainly that a binary
outside a clone has no first-party indexers, and how to get them from a clone.

## Decline: Container indexers in this plan
**Because (Jeff and Claude):** Jeff's idea, for the indexers and never for phoropter
itself, because getting every indexer working cross-platform may be hard. The contract
already allows it: an indexer is any program on `PATH`, so a `phoropter-index-go` that
runs scip-go in a container, with the checkout mounted, is just another indexer. But it
needs an image, a place to publish it, and a decision about old commits' toolchains,
which a container makes heavier, not lighter. `doctor` is where it would show up as an
option, once one exists. Whether it comes before or after the Windows trial is the
trial's to say.

## Decline: A piped install script
**Because (Jeff and Claude):** one line to install everything, which is why everyone
does it. But it runs code from the network that nobody's looked at, which is what Jeff's
CVE example was about. `doctor` gives the same help without that.

## Guess: `doctor` on a machine with nothing says everything that's missing, in one run
**Disproved if:** on a clean machine (the trial's), after `doctor`'s commands have been
run, anchoring a symbol in a language this repository uses still fails for a reason
`doctor` didn't name.

## Guess: Nobody needs the README to install the indexers
**Disproved if:** on the trial, Jeff or the fresh agent reaches for the README (or asks)
to get an indexer working, after running `doctor`.
