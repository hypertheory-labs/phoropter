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

You need Node 24 or later, and git.

```sh
git clone https://github.com/hypertheory-labs/phoropter.git
cd phoropter && npm install
ln -s "$PWD/src/cli.ts" ~/.local/bin/phoropter   # any directory on your PATH
```

`npm link` works too, unless a Node version manager keeps its global bin off your
`PATH`. Check it with `phoropter help`.

**Indexers**, for anchors on symbols (a class, a method) rather than files and quotes:
- **TypeScript:** included.
- **Go:** needs Go 1.25+ and `go install github.com/scip-code/scip-go/cmd/scip-go@latest`.
- **C# / .NET:** install `phoropter-index-roslyn` on your `PATH`. It's a separate
  repository with an `install.sh`.
- **Anything else:** see [`indexers/README.md`](indexers/README.md) to write one.

None are needed to start. File anchors with a quote work in any language.

## Use it on your repository

```sh
cd ~/code/your-repo
phoropter
```

It finds the repository's root, builds the site at `HEAD`, serves it on
`http://localhost:4173/` (or the next free port), and prints the address.
`phoropter url` prints it again, from anywhere in the repository. With no plans yet,
the home page says how to start.

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

**Symbol anchors:** `phoropter symbols <file>` lists a file's symbols, ready to paste
into a plan's `where:`. When plans anchor symbols, run `phoropter index` and commit
`.phoropter/index/`. Then the plans' anchors still resolve after the toolchain that could
index those old commits is gone.

## Tell your agent

Your repository's agent won't know about any of this unless its instructions say so.
Paste this into its `CLAUDE.md` (or `AGENTS.md`), and adjust it to taste:

```markdown
## Plans (phoropter)

This repository records its decisions as plans, in phoropter's format. Read the
guide first: it's at `guide/` on the site (`phoropter url` prints the address), or
`guide.md` in phoropter's repository.

- **Before non-trivial work, write a plan** in `plans/YYYY-MM-DD-<name>.md`: the
  decisions, the declines, and the guesses, each with its reason and an anchor to the
  code it's about. Commit it before the work.
- **When you're unsure, or I'd want to follow along, write a draft** instead:
  `drafts/plan.draft.md`, with open questions as prompts, each with what happens if
  nobody answers. Tell me it's up. My notes arrive in `drafts/plan.draft.notes.md`.
  Don't edit the draft while notes exist; take them in by rewriting it, then delete
  the notes. Convert it when we're confident. Silence means proceed.
- **After the work, write the as-built** beside the plan.
- **At the start of a session, look in `drafts/`** for notes on landed plans. Raise
  them before other work, and answer each in kind: a revisit, a citation in the
  current plan, or a reply.
- **Never edit a landed plan.** Changing an old decision is a revisit: a new plan, with
  `revisits:`, saying what moved.
- **To anchor a symbol,** `phoropter symbols <file>` lists the strings to paste.
- **You don't get a veto.** If you think an idea is wrong, say so up front in the
  draft, plan it properly anyway, and let me decide.
```

## The TUI

`tui/` is a terminal reader for the same site, written in Go. It's a projection of the
served pages, and it reads nothing else.

```sh
cd tui && go build -o ../bin/phoropter-tui .
bin/phoropter-tui "$(phoropter url)"
```

Run it inside the repository the site is about, so it can find the draft to annotate.
`?` lists the keys. [`tui/README.md`](tui/README.md) has more.

## Publishing

`phoropter build` writes the static site to `site/`. It holds plans and their code,
never drafts or notes. Publishing it, to GitHub Pages or anywhere static, is a seed in
`shelf/pages.md`.
