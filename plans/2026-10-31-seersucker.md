---
base: 5e923e8
---

# Indexers are programs in `indexers/`, with a guide for writing one

Phoropter indexes with scip-typescript and nothing else, so a symbol anchor in any other
language doesn't resolve. The first repository that wants one is this one: the TUI is
Go, and every plan written for it has used file anchors instead. Batik kept indexes per
indexer, made a missing one a `no-index` view, and loads an indexer only when something
needs indexing. This plan makes indexers pluggable.

It's the plugin plan's first half. Indexers and UIs are different contracts: an indexer
is a step in the build, chosen by the scheme an anchor names. A UI is a reader of the
pages, chosen by the person. UIs come next.

Jeff's test for it: he has code that indexes .NET with Roslyn, and he'd like to hand it
and this plan's guide to an agent and see what happens. So the contract can't assume
SCIP, and the guide has to stand on its own. An agent writing an indexer should never
need to open `src/`.

## Decision: An indexer is a program that prints definitions as JSON
- where (new): `indexers/README.md`

**Because (Jeff, then Claude):** "having a directory here with the go, typescript,
whatever." A program in any language can satisfy one plain contract:

- **In:** a directory holding the repository checked out at one commit, and the names of
  the packages the anchors ask for.
- **Out, on stdout:** one JSON object, in the store's shape. It names the indexer, its
  version, the scheme it serves, and for each package it found, every definition: the
  symbol (version written as `.`), file, first line, last line.
- **Packages it can't find** are listed as missing. A failure is a non-zero exit with one
  line on stderr. Phoropter turns either into `no-index` views, never a failed build.

SCIP indexers are adapters: they run the SCIP tool and turn its `.scip` into this JSON.
Roslyn, or anything else that knows where definitions are, can print it directly.

## Decision: The scheme an anchor names picks its indexer
- where: `src/scip.ts`
- quote: `export function isScipSymbol(where: string): boolean {`

**Because (Claude, not asked):** a symbol anchor already starts with its scheme
(`scip-typescript npm …`, `scip-go gomod …`). Symbols keep SCIP's grammar, scheme,
manager, package, version, descriptors, for every indexer, so anchors read alike in any
language. A scheme needn't start with `scip-`, so a Roslyn indexer can name its own
(`roslyn nuget …`). Phoropter groups the packages each document's anchors ask for by
scheme and runs each scheme's indexer once per commit for its packages.

## Decision: First-party indexers live in `indexers/`, others are found on `PATH`
- where: `src/site.ts`
- quote: `export function packagesOf(cx: Context, docs: Doc[]): Set<string> {`

**Because (Jeff, then Claude):** `indexers/<name>/` holds each first-party indexer, with an
`indexer.json` naming its scheme and the command that runs it. Any other indexer is a
program on `PATH` named `phoropter-index-<scheme>`, the way `gh` finds extensions, so
there's nothing to configure. Phoropter looks in `indexers/` first, then on `PATH`. Your
.NET indexer could live either way: in `indexers/dotnet/` if it joins this repository,
or installed on its own. "Good enough for gh and kubectl, good enough for us" (Jeff),
and `indexers/README.md` documents it. A config file in the repository was weighed.

## Decision: scip-typescript moves into `indexers/typescript/`
- where: `src/scip.ts`
- quote: `function scipTypescript(): { scip: any; main: string } {`

**Because (Claude, not asked):** the first indexer shows the contract, and it shouldn't be
special. What `src/scip.ts` does today (find the package roots by `package.json` name,
run scip-typescript, keep the definitions) becomes `indexers/typescript/`, a program
that prints the JSON. `src/` keeps only what every indexer shares: running one, reading
its output, and the symbol grammar. Mock-control's pages come out the same before and
after, which the tests check.

## Decision: A Go indexer in `indexers/go/`, so this repository's plans can anchor symbols
- where: `tui/go.mod`
- quote: `module phoropter-tui`

**Because (Claude, not asked):** a contract with one implementation isn't tested yet, and
the TUI is the case that's been waiting. `indexers/go/` finds a package root by its
`go.mod` module path (here, `phoropter-tui` in `tui/`) and prints its definitions. The
TUI's next plan can anchor `scip-go gomod phoropter-tui . internal/tui/…` and phoropter
shows it at the base, the as-built head, and now. It wraps scip-go: it runs it and turns
the `.scip` into the JSON. Its symbols are the ones Go's SCIP users already know,
Sourcegraph maintains it, and it's the fastest and cheapest way there (Jeff: "whatever
you think is fastest, cheapest"). It takes one `go install` on a machine that indexes
Go. Where it isn't installed, Go anchors show `no-index`. Its own program on
`go/packages` was weighed.

## Decision: The guide for writing an indexer is `indexers/README.md`
- where: `guide.md`
- quote: `Or a SCIP symbol, with the package version written as `.` so a release doesn't`

**Because (Jeff):** "a 'how to create an indexer' guide." It's written for an agent
handed only the guide and an existing piece of code: the contract, the symbol grammar
with examples in three languages, the JSON shape with a complete example, how package
roots are usually found, what to do about versions, and how to check the result. The
plan guide's section on symbol anchors links it, and says how to find a symbol's string.

## Decision: `phoropter index --check` tests an indexer against the contract
- where (new): `src/check.ts`

**Because (Claude, not asked):** "see how it goes" needs a way to tell. `phoropter index
--check <program> [path]` runs an indexer on a repository at `HEAD` and checks its
output: the JSON's shape, that every file exists at that commit, that every range is
inside its file, and that each definition's name appears in its range. It prints what
failed and why. An agent writing an indexer runs it until it passes, and so can a
reviewer.

## Decision: `phoropter symbols <file>` lists the symbols defined in a file
- where (new): `src/cli.ts`

**Because (Claude, not asked):** nobody types `scip-go gomod phoropter-tui .
internal/tui/docScreen#annotate().` from memory, an agent included. `phoropter symbols
tui/internal/tui/screens.go` prints each symbol defined there with its lines, from the
store, indexing if it must, and ready to paste into a plan's `where:`. It's how anchors
get written in a language the writer hasn't anchored in before.

## Decline: A .NET indexer, in this plan
**Because (Jeff):** that's the test of the guide, not part of the work. An agent given
Jeff's Roslyn code and `indexers/README.md` writes it, after this lands.

## Decline: Changing the store's layout for schemes
**Because (Claude, not asked):** batik keys a stored index by commit and package. Two
schemes could name the same package, but each stored file already names its indexer and
scheme, so a mismatch reads as missing and is indexed again rather than misread. Adding
the scheme to the path would revisit batik for a collision nobody has had.

## Guess: An agent with only `indexers/README.md` and existing code writes an indexer that passes `--check`
**Disproved if:** given Jeff's Roslyn code and the guide, the agent has to read `src/`,
asks something the guide should have answered, or ends with an indexer that `phoropter
index --check` fails on a .NET repository.
