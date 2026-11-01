# Seed: lenses, a reader's own subsets of plans

From Jeff, 2026-11-01, after a day on the course repository with phoropter. Two
observations started it:
- **The agent grepped `plans/`** instead of using the site. That's rational: the plans
  are local markdown and grep is one call. What grep misses is the relations the site
  computes (answered, revisited, drifted, what else points here). "Grep to find, the
  site to understand" may be the split. Worth asking an agent why it greps before
  deciding anything.
- **Jeff wanted search in the TUI**, for "wait, why did we defer that again?" in the
  middle of a day's work. But the plan probably says "Decline" or "waits", not "defer",
  so text search misses. Jeff: search is "everything else failed". It'll probably come,
  as the grep answer exposed through the TUI, maybe from one page listing every entry
  as microdata (`shelf/draft-places.md` floated it). Collect real search questions as
  they come up; the plan starts from those.

## A lens

A named subset of plans and entries that one reader keeps: "auth", or "what this work
should keep in mind". A lens plus a kind filter ("auth", only Declines) answers most
"why did we…" questions without search. It's a reader's lens, never the plan's claim,
so plans stay testimony. Landed plans never change, so a lens's references can't come
loose.

- **Plans or entries,** both.
- **Agents can make them:** "mark the plans that intersect with what we're doing" is
  judgment by meaning. It catches the case anchors can't: a new plan that confuses an
  old one about an idea, with no shared place.
- **The format is the anchor syntax a plan already uses,** one reference per line,
  `plans/<file>.md` or `plans/<file>.md › <entry title>`, under a header naming the lens
  and the repository by its first commit's sha. A person reads it without phoropter, so
  it can be pasted into a ticket or a message to a coworker: "keep these earlier plans in
  mind". Importing a lens is pasting it.
- **Two homes, one format:** durable lenses in personal config outside the repository
  (keyed by the first commit, which every clone shares; lenses are never cross-repo); a
  session lens in `drafts/`, beside the draft, ignored or committed as `drafts/` is.
- **All personal.** Sharing is pasting, so there's nothing to sync. Teams need nothing
  more.
- **Notes on a lens:** a personal note that lights up when you come back ("you have a
  note about this"). It has no state (no open or done, no date, no assignee), so it
  can't become a backlog or a tracker. That's the line. Notes on landed plans are
  different: they're addressed to the agent and answered in kind.
- **"mine"** isn't a useful lens when working solo, since every commit is yours. On a
  team, git knows. Solo, it's just a lens of plans you care about.

To weigh: where the dev server and the TUI show lenses (never on a built site, like
notes); whether a lens can also hold places (a file, a symbol); and how this meets file
mode (`shelf/file-pages.md`), where the stack of plans touching a file is a lens that
phoropter computes.

**Jeff, later the same day: file view version 1 is a lens.** Paste a relative path, and
the TUI makes a lens of every plan and entry that touches that file. It keeps the guard
in `shelf/file-pages.md`: you bring a path from somewhere (a stack trace, a diff, the
agent), so it's a question, not browsing a tree. It needs two gaps closed so the lens is
complete: a declared way to find a file's place from its path (`place-url-convention`),
and the symbol anchors inside a file listed on its page (`symbol-file-unlinked`).
Otherwise anchors on a class in that file are missing.

**Jeff, once lenses existed (percale):** a lens is a way to start a conversation. Collect
a few entries, add a note ("the new version of super-framework-2000 might change some of
this, want to talk it through?"), and hand it to the agent. The lens stays stateless; what
comes out of the conversation is testimony in the usual places: a revisit of the plan
whose ground moved, a citation in the current draft, or nothing. Close to a note on a
landed plan, but across several plans at once, and about an idea, not one plan.
