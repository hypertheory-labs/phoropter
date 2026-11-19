# Writing an indexer for phoropter

Phoropter shows a repository's plans with the code they're about. A plan's entry can
anchor a symbol, like a class or a method, and phoropter shows that symbol at the
commit the plan was written against, where the plan's own work landed, and now. To find
a symbol at a commit it needs an **indexer**: a program that reads a repository checked
out at one commit and says where every definition is.

This page is the whole contract. You can write an indexer from it and an existing piece
of code that understands your language (a compiler API, a language server, a SCIP
indexer) without reading phoropter's source. When `phoropter index --check` passes, it's
done.

Two indexers live beside this page, as examples:
- `typescript/` wraps scip-typescript.
- `go/` wraps scip-go, and fills in where each declaration ends from Go's own parser.

## The contract

An indexer is a program. Phoropter runs it two ways.

### `<program> --describe`

Print one JSON object and exit 0:

```json
{ "indexer": "phoropter-index-go", "version": "0.1.0", "scheme": "scip-go", "extensions": [".go"] }
```

- **`indexer`:** its name.
- **`version`:** its version. Put the version of anything it wraps in here too, like
  `"0.1.0 (scip-go 0.2.9)"`. Stored indexes name it.
- **`scheme`:** the first word of every symbol it produces. Anchors starting with this
  word are sent to this indexer. See below.
- **`extensions`:** the file extensions it indexes. `phoropter symbols <file>` picks
  indexers by these.
- **`needs`** (optional): what it couldn't find to run, each as `{ "what": …,
  "install": … }`, the thing and the command that would get it. Leave it out when
  nothing is missing, as far as you know. `phoropter doctor` shows it to whoever is
  setting up, so check what you can cheaply (a tool on `PATH`, a package installed)
  and don't index anything to find out. Still exit 0: an indexer that needs something
  has still described itself.

  ```json
  { "indexer": "phoropter-index-go", "version": "0.1.0", "scheme": "scip-go", "extensions": [".go"],
    "needs": [{ "what": "scip-go", "install": "go install github.com/scip-code/scip-go/cmd/scip-go@latest" }] }
  ```

  `needs` is about running at all. Packages that couldn't load at one commit go in
  `missing`, in the output below.

### `<program> <dir> [package...]`

`<dir>` is the repository checked out at one commit, at a temporary path. Each
`package` is a package name, as it appears in symbols. With no packages named, index
every package you find, anywhere in `<dir>`: a repository may hold several solutions,
workspaces or modules. Print one JSON object on stdout and exit 0.

**The checkout is bare.** Nothing has been restored, installed or built, and the
network may not be there. The commit may be years old, and its packages, SDK or
toolchain may be gone. An indexer that works from source alone (parsing, not
compiling) keeps working on old commits; one that needs a restore or a build fails
where it can't get them, or takes minutes where it can. A .NET experiment measured it:
a syntax-only indexer took 0.2s on any commit, offline, and a build-based one took 8 to
52s and failed or lost projects without the network. If yours needs a restore, make it
optional, and report what couldn't load in `missing`.

```json
{
  "indexer": "phoropter-index-go",
  "version": "0.1.0 (scip-go 0.2.9)",
  "scheme": "scip-go",
  "packages": {
    "phoropter-tui": {
      "definitions": {
        "scip-go gomod phoropter-tui . `phoropter-tui/internal/tui`/docScreen#": ["tui/internal/tui/screens.go", 260, 271],
        "scip-go gomod phoropter-tui . `phoropter-tui/internal/tui`/docScreen#annotate().": ["tui/internal/tui/screens.go", 319, 344]
      }
    }
  },
  "missing": ["some-package-it-couldnt-find"]
}
```

- **`packages`:** for each package found, every definition: the symbol, then
  `[file, firstLine, lastLine]`.
  - The file is relative to `<dir>`, with `/` separators, and must be a file that was in
    the checkout. Generated files (`obj/`, build output, source-generator output) and
    cached files elsewhere on the machine aren't definitions: no plan can quote them.
  - Lines are 1-based and inclusive.
  - The range is the **whole declaration**, from its first line through the end of its
    body, not just the line its name is on. Phoropter searches for a plan's quote within
    this range. A one-line range means a quote inside a method is never found.
- **`missing`:** every package you know of but couldn't index, whether or not it was
  named: one you couldn't find, and one you found but couldn't load. Never leave a
  failed package out, or it looks like it doesn't exist. Phoropter shows its anchors as
  `no-index`. It may be omitted when empty.
- **Failure:** exit non-zero and write one line to stderr saying why, for example
  `scip-go isn't installed: go install …`. Phoropter puts that line on the anchor's
  view and builds everything else. Never let a failure look like an empty result.

## Symbols

Every symbol uses [SCIP's grammar](https://github.com/sourcegraph/scip/blob/main/scip.proto),
whether or not your indexer uses SCIP. That way anchors read alike in every language:

```
<scheme> <manager> <package> <version> <descriptors>
```

| Language | Example |
|---|---|
| TypeScript | ``scip-typescript npm @hypertheory-labs/mock-control . src/lib/panel/`mock-panel.ts`/MockPanel#`` |
| Go | ``scip-go gomod phoropter-tui . `phoropter-tui/internal/tui`/docScreen#annotate().`` |
| C#, say | `roslyn nuget Contoso.Orders . Contoso/Orders/OrderService#Place().` |

- **`scheme`:** names your indexer. If you wrap a SCIP indexer, use its scheme. If not,
  pick a short lowercase name of your own. It's written into plans forever, so don't
  change it later.
- **`manager`:** the package manager: `npm`, `gomod`, `nuget`, `maven`, `cargo`.
- **`package`:** the name a package is known by. For npm, the `name` in its
  `package.json`. For Go, the module path in `go.mod`. For .NET, the project's
  `AssemblyName`, or the project file's name when it isn't set: the compiler's name for
  it, readable without loading anything. Phoropter asks for packages by this name.

  A package name may not be unique in a repository: copies of a project in several
  solutions often share one. Index each project on its own, and where two define the
  same symbol, the first in path order wins. Never number overloads or otherwise
  combine across copies; that invents symbols that exist nowhere. The copies that lose
  show themselves: `phoropter symbols` on one finds nothing, and names the file at the
  first path that was indexed instead.
- **`version`:** always `.`. Plans store symbols without versions, so a release doesn't
  break an anchor. Replace whatever version your tool produces with `.`.
- **`descriptors`:** the path to the thing, each part ending in a suffix that says what
  it is:

  | Suffix | Kind | Example |
  |---|---|---|
  | `/` | namespace, package, or directory | `Contoso/Orders/` |
  | `#` | type: class, struct, interface, enum | `OrderService#` |
  | `.` | term: field, property, constant, variable | `OrderService#total.` |
  | `().` | method or function | `OrderService#Place().` |
  | `(+1).` | an overload of a method, numbered from 1 | `OrderService#Place(+1).` |
  | `[T]` | type parameter | `Repository#[T]` |
  | `(x)` | parameter | `OrderService#Place().(order)` |

  A name with characters other than letters, digits, `_`, `+`, `-` and `$` goes in
  backticks: ``src/`mock-panel.ts`/``.

Leave out local variables. A plan anchors what a reader can find by name. Every type,
member, function, field and constant is a definition. Parameters and type parameters
are optional, and better left out.

### Naming what the table doesn't cover

**When your language already has a SCIP indexer, name things as it does**, even if you
don't wrap it: constructors, indexers, operators, events, accessors and the like have an
established spelling there, and anchors read alike across tools. Read its source or its
test output to find it. **Except where it loses information**, because a symbol that
collides points an anchor at the wrong thing:
- **Keep full namespace paths.** scip-dotnet keeps only the innermost segment
  (`Vendors/VendorController#` for `Software.Api.Vendors.VendorController`), so types
  that share a last segment collide. Write `Software/Api/Vendors/VendorController#`.
- **Write generic arity when a name is reused.** If `Foo`, `Foo<T>` and `Foo<T,U>` share
  a namespace, the lowest arity is `Foo#` and the others use the language's own arity
  spelling, like .NET's `` `Foo`1`# ``.

**A type declared in several places** (a partial class across files) gets one definition:
the first part, ordering files by path, then by position. Its members keep their own
ranges, so a quote inside one is still found. Number overloads in that same order.

**Code with no name a reader can find** (top-level statements, the compiler's implicit
`Program` and `<Main>$`) gets no definition. A plan anchors that file with a file anchor
and a quote instead.

## Installing an indexer

Phoropter looks for the indexer for a scheme in two places, in this order:

1. **`indexers/<name>/` in phoropter's own repository**, for first-party indexers. The
   directory holds an `indexer.json` naming the command, run from that directory:
   ```json
   { "command": ["go", "run", "."] }
   ```
2. **A program on `PATH` named `phoropter-index-<scheme>`**, for any other indexer. That's
   how `gh` and `kubectl` find extensions, and there's nothing to configure. Install it
   anywhere on your `PATH`, make it executable, and phoropter finds it by its scheme.
   For example, `phoropter-index-roslyn` serves symbols starting with `roslyn`.

`phoropter indexers` lists every indexer phoropter finds, and where it found each one.

A machine without an indexer for a scheme still builds every page. That scheme's
anchors show `no-index`, naming what's missing.

## Checking it

```sh
phoropter index --check indexers/dotnet     # a directory with an indexer.json
phoropter index --check ./my-indexer        # or any program
```

Run it inside a repository written in your language. It runs `--describe`, then runs
the indexer on the repository at `HEAD` with no packages named, and checks that:
- the output has the shape above, in the scheme `--describe` gave;
- every symbol is in the grammar, under the package it names, with version `.`;
- every file exists at `HEAD`;
- every range is inside its file;
- each definition's name appears in its range. A backticked name is read whole
  (`` `.ctor` `` is `.ctor`). Only names a source file spells as written are looked for,
  so namespaces (named by their paths) and names an indexer makes up (`.ctor`,
  `this[]`, `<constructor>`) are skipped. Up to 5% of names may be missing from their
  ranges, for names a language spells differently, like an operator's `op_Addition`;
  they're listed as notes. More than that is a range bug, and fails.

It prints what failed and why, and exits non-zero until everything passes. Both
indexers here pass it on phoropter's own repository.

## How phoropter uses what you print

- Phoropter runs the indexer on a temporary checkout, once per commit, for the packages
  a site's anchors ask for. **It gives it 5 minutes** (`PHOROPTER_INDEX_TIMEOUT`, in
  seconds, changes that), then stops it and shows its anchors as `no-index`, timed out.
  An indexer that works from source finishes in well under a second on most
  repositories.
- It keeps the definitions, never re-indexing a commit it already has:
  - the commits plans were written against go in `.phoropter/index/<sha>/<package>.json`,
    committed by `phoropter index`, so an old plan's anchors resolve after your
    toolchain for that era is gone;
  - everything else goes in a local cache.
- Your output becomes those files directly, so keep it to definitions. Sort nothing,
  phoropter sorts.

## Writing anchors once it works

```sh
phoropter symbols tui/internal/tui/screens.go
```

This prints each symbol defined in the file with its lines, at `HEAD`, ready to paste
into a plan. When more than one installed indexer handles the file's extension, each
scheme's symbols are listed under its own heading; `--scheme <scheme>` shows one. The
scheme a plan's anchor starts with decides which indexer finds it.

```markdown
- where: ``scip-go gomod phoropter-tui . `phoropter-tui/internal/tui`/docScreen#annotate().``
- quote: `func (s *docScreen) annotate(m *Model) {`
```

Use a double-backtick code span when the symbol contains backticks, as the plan guide
says.

## Starting from a compiler API

If you're starting from a compiler or language service, not a SCIP indexer, the
shape is usually:

1. **Find the packages:** walk `<dir>` for your project files, and read each one's
   name.
2. **Load each package** with your tool's own loader. Roslyn's workspace, Go's
   `go/packages`, and the TypeScript compiler API all do this.
3. **Visit every declaration** in the package's own source files. For each one, take
   the symbol the compiler gives it, and the span of the whole declaration node, and
   turn the span into 1-based lines.
4. **Build the symbol string** from the namespace and type containment, using the
   suffixes above. Number overloads in declaration order.
5. **Print the JSON,** and run `phoropter index --check` until it passes.
