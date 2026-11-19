---
head: 8764b8e
---

# As built: grosgrain

Built as planned, with one change of name and three things the plan didn't foresee,
all found by running `doctor` the way a newcomer would. `go test -count=1 ./...` passes,
and `phoropter index --check indexers/go` passes.

The newcomer was simulated three ways: a copy of the binary outside any clone, with a
`PATH` of only `/usr/bin` and `/bin` and an empty home; the same, pointed at a clone's
indexers; and the same, with a built Go indexer on `PATH`. The first version of `doctor`
got the first and third right and the second wrong. Its advice for a missing Go indexer
was "an indexer named phoropter-index-scip-go on `PATH`", which is true to the contract
and no help to anyone. Each case now gets the fix it actually needs, in the order a
newcomer meets them: clone phoropter, then install Go, then `go install` scip-go. A test
holds all three.

`phoropter index --check indexers/typescript` fails on phoropter's own repository, as it
has since organdy removed phoropter's last TypeScript project (poplin's as-built). The
test suite checks that indexer against a TypeScript repository it builds, and that
passes.

## Changed: The field is `needs`, not `missing`
- where: ``scip-go gomod github.com/hypertheory-labs/phoropter . `github.com/hypertheory-labs/phoropter/internal/site`/IndexerInfo#Needs.``
- quote: ``Needs   []Need   `json:"needs,omitempty"` ``

**Because (Claude, not asked):** the contract already has a `missing`, in an indexer's run
output: the packages that couldn't load at one commit. The same word in `--describe` for
"the tools I need to run" would make one contract word mean two things. Each need is
`{ "what": …, "install": … }`, and `indexers/README.md` says how they differ.

## Chose: Phoropter knows its own two indexers' toolchains
- where: ``scip-go gomod github.com/hypertheory-labs/phoropter . `github.com/hypertheory-labs/phoropter`/firstPartySchemes.``
- quote: `"scip-go":         {"Go", "go", "Go 1.25 or newer (https://go.dev/dl)"},`

**Because (measured, then Claude, not asked):** an indexer that can't start can't describe
its needs. Phoropter's Go indexer is `go run .`, so without Go there's nobody to say
"install Go". The plan put that knowledge in the indexer, which is right for everything
an indexer can say once it runs. For the two that ship in `indexers/`, phoropter names
the toolchain, and where to get them when there's no clone. Anyone else's indexer still
speaks for itself.

## Chose: Indexers that can't start are remembered, not dropped
- where: ``scip-go gomod github.com/hypertheory-labs/phoropter . `github.com/hypertheory-labs/phoropter/internal/site`/IndexersUnavailable().``
- quote: `func IndexersUnavailable() []Unavailable {`

**Because (Claude, not asked):** `Indexers()` silently skipped an indexer whose
`--describe` failed, so a machine without `node` just had no TypeScript, and nothing
said why. Now it keeps them with their error, and `doctor` prints each one.

## Chose: `doctor` exits 1 when something is missing
- where: ``scip-go gomod github.com/hypertheory-labs/phoropter . `github.com/hypertheory-labs/phoropter`/doctorFacts#report().``
- quote: `func (d doctorFacts) report() (string, bool) {`

**Because (Claude, not asked):** an agent or a script can tell "ready" from "not ready"
without reading the text. A repository whose plans anchor no symbols is ready, and
`doctor` says file anchors work in any language.

## Chose: `--check` validates `needs`, and says a run may fail because of them
- where: `internal/site/check.go`
- quote: `problems = append(problems, "--describe has a need without both what and install, so phoropter doctor can't say how to fix it")`

**Because (Claude, not asked):** the plan said `--check` checks the shape when it's
there. A need without a command is one `doctor` can only half say. And an indexer that
reports a need will probably fail the run below it, so `--check` says that first.

## Answered: `doctor` on a machine with nothing says everything that's missing, in one run
**Answer (Claude):** not tested on a clean machine, which is the trial's job. It held
in the three simulations above, once the second was fixed. A real clean machine has
things a simulation doesn't: no Homebrew, a different shell, Windows paths. The Windows
VM will have plenty.

## Answered: Nobody needs the README to install the indexers
**Answer (Claude):** not tested. It needs the trial.
