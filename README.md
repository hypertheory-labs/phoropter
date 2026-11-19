# phoropter

Read code through the plans that shaped it. A person and an agent write a short plan
before each piece of work: what they decided, what they turned down, what they're
guessing, each with its reason and the code it's about. Phoropter turns a repository's
plans into a site where each decision sits beside its code, as it was then and as it is
now. The same pages are the API an agent reads, so the person and the agent see the
same picture.

The plan format itself is in [`guide.md`](guide.md), and every site serves it at
`guide/`. This page is how to use phoropter on a repository of your own.

## Install

phoropter is one Go binary, with the terminal reader inside. You need git, and Go 1.25+
to build it.

```sh
go install github.com/hypertheory-labs/phoropter@latest
```

That binary anchors files and quotes, but not symbols, until it has an indexer. The Go
indexer installs the same way (below). For the TypeScript one, install from a clone:

```sh
git clone https://github.com/hypertheory-labs/phoropter.git
cd phoropter && go build -o bin/phoropter .
ln -s "$PWD/bin/phoropter" ~/.local/bin/phoropter   # any directory on your PATH
```

Check it with `phoropter help`. `phoropter guide` prints the plan format, and
`phoropter indexers` lists the languages it can index here. In a repository,
`phoropter doctor` says what this machine is missing for the languages its plans
anchor, and the command that would fix each. It runs none of them.

**Indexers**, for anchors on symbols (a class, a method) rather than files and quotes.
`phoropter doctor` says which of these a repository needs:
- **TypeScript:** in a clone, Node 24+ and `npm install` (it runs scip-typescript).
- **Go:** Go 1.25+, scip-go (`go install github.com/scip-code/scip-go/cmd/scip-go@latest`),
  and, outside a clone, `go install github.com/hypertheory-labs/phoropter/cmd/phoropter-index-scip-go@latest`.
- **C# / .NET:** install `phoropter-index-roslyn` on your `PATH`. It's a separate
  repository with an `install.sh`.
- **Anything else:** see [`indexers/README.md`](indexers/README.md) to write one.

None are needed to start. File anchors with a quote work in any language.

## Use it on your repository

```sh
cd ~/code/your-repo
phoropter init   # once: your first plan, as a draft, and what to tell your agent
phoropter
```

`phoropter init` prints what to do next. After that, `phoropter` finds the repository's
root, builds the site at `HEAD`, serves it on `http://localhost:4173/` (or the next free
port), and prints the address. `phoropter url` prints it again, from anywhere in the
repository. As commits land, the site follows `HEAD`. In a terminal, `h` + enter lists
keys to rebuild, open the browser, print the address, or quit. Usually your agent runs
it, in the background, and one server lasts the session. With no plans yet, the home
page says how to start.

A day's loop:

1. **Before a piece of work, write a plan** in `plans/YYYY-MM-DD-<name>.md`. The date is
   the day you say "we're implementing this", and the name is any word that means
   nothing. Usually the agent writes it. Commit it before the work starts. `guide.md`
   says what goes in it.
2. **When you want to think it through first, make it a draft:** `drafts/plan.draft.md`,
   one at a time. Open it at `draft/` on the site, or in the TUI. Annotate it in the
   TUI (`a` on an entry or a prompt), or in an editor that speaks the annotating
   agreement. Notes go in `drafts/plan.draft.notes.md`. When it's ready, the draft
   converts into a plan, and the draft and its notes are deleted.
3. **Do the work.**
4. **Write the as-built:** `plans/YYYY-MM-DD-<name>.as-built.md`, saying what changed
   from the plan, what was chosen along the way, and how each guess turned out.
5. **Later, when an old plan matters to something new,** write a note on it (`a` in the
   TUI, on a landed plan). It's kept in `drafts/`, never in the plan. Changing an old
   decision is a revisit: a new plan that cites the old one.

**`drafts/` in git:** ignore it if you work alone or as a pair on main. Commit it on a
branch when a teammate or a mentor should read a draft, or your notes on an old plan,
before anything is decided.

**Lenses:** a lens is your own subset of plans and entries, like "auth" or "what this
work should keep in mind", kept in `drafts/lenses/<name>.md`. It's a title and one
reference per line, the syntax a plan's `where:` uses, each optionally followed by a note
to yourself:

```markdown
# Auth

- ``plans/2026-10-31-batik.md › `phoropter index` is the one command that writes into the repository``
- `plans/2026-11-01-poplin.md` — the whole plan
  - lines indented under an item are its note, in markdown
  - handy for prompt prep: what you want to talk through, and why

Prose after the list is the lens's own closing note.
```

The dev server shows them at `lenses/`. In the TUI, `L` lists them, `f` filters one by
kind (Decisions, Declines, open guesses), `+` adds what you're on to one, and `e` opens
one in your editor. A lens can be pasted into a ticket or a message as is. A built site never has lenses: they're a way
of reading the plans, not part of them.

**Symbol anchors:** `phoropter symbols <file>` lists a file's symbols, ready to paste
into a plan's `where:`. When plans anchor symbols, run `phoropter index` and commit
`.phoropter/index/`. Then the plans' anchors still resolve after the toolchain that could
index those old commits is gone.

## Tell your agent

Your repository's agent won't know about any of this unless its instructions say so.
`phoropter init` writes your repository's first plan, "How we work here", as a draft:
the conventions this README used to list (when to write a plan, when a draft goes up,
who runs the server, lenses, no veto), each as a Decision with phoropter's default.
You and your agent work that draft like any other, and change what doesn't suit you.

It also prints a pointer to paste into `CLAUDE.md` (or `AGENTS.md`). It doesn't edit
them itself:

```markdown
## Plans (phoropter)

This repository records its decisions as plans. `phoropter guide` prints the
format. The first plan, "How we work here" (the oldest in `plans/`, or
`drafts/plan.draft.md` until it converts), says how we work. Read both before
other work.
```

Only the pointer is pasted, so only the pointer can go stale, and it names nothing that
changes: the format comes from the binary, and the conventions from your own first plan.
If you pasted the longer block this section used to hold, replace it with the pointer
and run `phoropter init`.

## The TUI

`phoropter ui` is a terminal reader for the same site. It's a projection of the served
pages and reads nothing else; a test keeps it from importing phoropter's own code. It
opens the site `phoropter url` finds for the repository you're in, or the address you
give it. Run it inside the repository the site is about, so it can find the draft to
annotate. `?` lists the keys. [`tui/README.md`](tui/README.md) has more.

## Publishing

`phoropter build` writes the static site to `site/`. It holds plans and their code,
never drafts or notes. Publishing it, to GitHub Pages or anywhere static, is a seed in
`shelf/pages.md`.
