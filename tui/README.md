# Phoropter TUI 

I'd like to explore an experiment in creating a TUI (probably Go?) for a human exploring the substrate presented at http://localhost:4173/ right now.

It's REST, microdata, etc. The repo for that tool is at ../phoropter and it's generating that HTML from ../mock-control 

We are in the early stages of building this. So part of what I'm looking for here is to identify shortcomings in the HTML API. 

## Roughly, what I'm envisioning 

A tool that will be used in the context of a repo that is serving the substrate. A list of the plans will be presented. Developer can select a plan, see the details, and for each step, see (maybe in another pane) the code - so the source code in "layers" somehow - maybe navigable by keyboard shortcuts - what it was at the time of the plan under observation, what it is now, and a list of other plans that involve this file.

This will be completely read only. There is an "edge" feature coming that will have some write ability, but never to the substrate itself.

## Running it

The TUI is part of the `phoropter` binary (organdy):

```sh
phoropter ui                                     # the site phoropter url finds, or http://localhost:4173/
phoropter ui tui/testdata/phoropter-1a83dbc      # or a built site on disk
phoropter ui gaps [site]                         # crawl, and list where it read past the profile
go test ./...                                    # from the repository's root
```

Reasons come only from the substrate's pages; the TUI reads no git history. It writes
only notes, in `drafts/`: a draft's, and notes on landed plans. Run it inside the repository the site is about, so it
can find the draft.

- **Plan list:** the draft slot on top, then plans by title, newest first. `j`/`k` or
  `tab`/`shift+tab` move, `o` flips the order, `enter` opens.
- **A plan:** `1` `2` `3` flip an anchor between before, as built, and now. `n` cycles
  anchors, `enter` follows the rels out of an entry.
- **A landed plan:** `a` writes a note on an entry or, on About, on the whole plan, to
  say it matters now. The note goes in `drafts/`; the plan is never edited.
- **A draft:** `a` answers a prompt, notes an entry, or on About notes the whole draft.
  `s` skips a prompt on purpose. In a form, `ctrl+s` writes, `tab` moves between fields,
  `ctrl+e` opens `$EDITOR`, and `esc` cancels.
- **Anywhere:** `r` reads the page again, `g` shows the gaps hit so far, `?` is help.
  `y` copies the address of what you're on (a plan's at the entry), `b` opens it in the
  browser, and `Y` copies its reference, `plans/<file>.md › <entry title>`, which
  outlasts the server's port (nainsook).

To try a draft: from the repository root, `mkdir -p drafts && cp
tui/testdata/sample.draft.md drafts/plan.draft.md`, run `phoropter`, and open the TUI
with `phoropter ui`, which finds the server itself.

The TUI lives in phoropter's repository as `tui/` since dowlas (the TUI moves in). Its
plans are in the root `plans/` with phoropter's. Where it's going is in `docs/next.md`,
and how notes are written is in `docs/annotating.md`. What it asked phoropter for
before the move is in `docs/for-phoropter.md`, superseded by seeds in the root
`shelf/`.

`testdata/` holds sites built by phoropter (`phoropter build <repo> --at <commit> --out
…`) from mock-control at `f02528f` and phoropter at `1a83dbc`, both built by phoropter at
`1a83dbc`, so tests don't need a server. Rebuild them when the profile changes.
`mock-control-draft/` is a draft page from phoropter's dev server, and
`mock-control-draft.source.md` is the draft it was served from.
