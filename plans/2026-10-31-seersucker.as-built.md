---
head: 32f2189
---

# As built: seersucker

Tried end to end: a throwaway draft anchored
``scip-go gomod phoropter-tui . `phoropter-tui/internal/tui`/docScreen#annotate().``
with a quote from its body, and phoropter's draft page showed it found, "lines 319–344;
quote at line 332". The draft was taken down after. Mock-control's pages are unchanged
by moving scip-typescript out of `src/`: the tests that check its symbol views pass as
they did.

## Changed: scip-go is pinned to v0.1.23
- where: `indexers/go/main.go`
- quote: `fail("scip-go isn't installed: go install github.com/sourcegraph/scip-go/cmd/scip-go@v0.1.23 (newer versions need a newer Go)")`

**Because (Claude, measured):** the plan said one `go install`. The latest scip-go needs
Go 1.26, and Jeff's machine has 1.24.2. v0.1.23 is the newest that runs on 1.24.0 or
later. The install message, CLAUDE.md and the guide say so.

## Chose: The Go indexer fills in where a declaration ends
- where: `indexers/go/main.go`
- quote: `func declEnds(path string) map[int32]int32 {`

**Because (Claude, measured):** scip-go v0.1.23 records where a definition's name is, not
where the declaration ends, so every function came out one line long, and a quote in its
body was never found. Go's own parser knows where each declaration ends, from the
standard library, with no new dependency. The guide makes it part of the contract that
a range is the whole declaration.

## Chose: `--check` takes an indexer's directory as well as a program
- where: `src/cli.ts`
- quote: `// A directory with an indexer.json (indexers/go, or the one you're`

**Because (Claude, measured):** `go run` has to run inside its module, so checking the Go
indexer by command failed. Given a directory with an `indexer.json`, `--check` runs its
command there. That's also how an agent building `indexers/dotnet/` would check it in
place.

## Chose: `--check` knows the names indexers invent
- where: `src/check.ts`
- quote: `function spellings(name: string): string[] {`

**Because (Claude, measured):** its first run failed both first-party indexers on names
no source spells: scip-typescript's `<constructor>` and its numbered properties of
anonymous objects (`encoding0`), and every namespace, whose name is a path. It accepts
those, and still fails a name that should be in its range and isn't. Its first run also
caught a real bug: scip-go indexes Go's synthetic test packages, whose files are in Go's
build cache. The Go indexer now drops files outside the repository.

## Answered: An agent with only `indexers/README.md` and existing code writes an indexer that passes `--check`
**Answer (Claude):** not tested. That's Jeff's experiment, with his Roslyn code and a
fresh agent. Writing the guide doesn't test it: the agent who wrote the guide has read
`src/`.
