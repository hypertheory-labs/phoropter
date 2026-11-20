# Seed: lenses, what's still open

Percale and lawn landed lenses: `drafts/lenses/<name>.md`, served at `lenses/`, listed,
filtered and collected (`+`) in the TUI, with notes, and opened in your editor (`e`). The
reasoning is in those plans. What's still open, from Jeff:

- **A durable home.** A lens starts in `drafts/` (Jeff's choice in percale). Saving one
  to personal config outside the repository, keyed by its first commit, so it outlasts a
  fresh clone, is a step someone takes on purpose. Not built.
- **Lenses that hold places,** a file or a symbol, not only plans and entries. Nainsook
  made `Y` on a place copy what an anchor at it would say, which is what such a line
  would contain. File view version 1 is a lens made from a pasted path
  (`shelf/file-pages.md`).
- **"You have a note about this,"** shown on a plan's page when a lens holds it. Percale
  declined it for the first version.
- **Set operations.** A lens is a set of references, so lenses combine: "auth ∩ cms",
  "auth − what we covered in class". The kind filter is already one. TUI keys for them
  wait "until Jeff catches himself wanting one".
- **Search.** Jeff: search is "everything else failed". Collect real search questions as
  they come up, and the plan starts from those. The agent grepping `plans/` instead of
  the site may be the right split: "grep to find, the site to understand".
- **A shelf home for lenses** (2026-11-01): a seed reads well as a lens, but the shelf is
  committed and shared, and a second lens home is a bigger step than a trial showed the
  need for.

**Code spans, and polads** (Jeff and Claude, 2026-11-20, from the trial's VS Code
extension work):

- **A lens is a set of specimens:** gathered to help draft a plan, pinned, likely to
  drift, discarded by default. So a lens can hold code, not only plans: a span is the
  anchor syntax a plan already uses (a file and a quote, or a symbol), never line
  numbers. It's read at the working tree; a quote that no longer matches says so, and
  that drift means the lens is old enough to throw away. The commit that matters is the
  `base:` of the draft it feeds.
- **Harpoon, not smart folders.** One active lens at a time, saved and switched between,
  never shown as folders in an explorer. Jeff recoiled at "defined by a query": a smart
  folder is a query nobody chose the contents of. So `affected` may print a lens, as a
  snapshot to prune, and never refreshes one.
- **Not Code Tour:** its failure is the labor. Adding a span costs what `+` costs, one
  key on a selection or one line from the agent, with the note optional and later.
- **Portable by being Markdown:** pasted in a chat, mailed, embedded in a doc. They live
  in `drafts/lenses/` so they're ephemeral by default; nothing stops copying one out,
  but phoropter doesn't make them a thing.
- **"Which plans touch this span?"** on a code item, the overlapping anchors' plans: what
  makes a span more than a bookmark. `affected` at the size of a span.
- **A polad is a speculative change**, in `drafts/` too. Two uses:
  - *Teaching:* a scratch projection of an idea, close to the real code's meaning but
    written to get something understood, not to compile. This is where the device came
    from, and it's the guard on the other use.
  - *Blast radius:* "we're switching tokens to cookies; show me what it touches and
    what it does to earlier plans." A polad as a commit on a throwaway branch or
    worktree, with every anchor resolved there and compared with HEAD: which Decisions'
    code moves or vanishes, which Declines sit by changed code, which open guesses it
    would test. `affected --since` from the polad's side.
  - **A polad is never applied.** It informs the draft and is discarded; the work is
    redone under the plan, so the plan isn't a caption on code already written.

**Lenses over time** (Jeff and Claude, 2026-11-20): "what would the Windows lens have
looked like in January 2023?" Two questions. The same lens read at an old commit
(`--at`) is mechanical: which items weren't plans yet, which anchors were where. The lens
someone would have built then, by an agent limited to that commit's plans, compares two
judgments; phoropter can show them side by side and leave the reading. Topic lenses
replace tags (`premises.md`); a demonstration is `drafts/lenses/windows.md`, where one
entry grep missed (nainsook's `clip`) was found by asking what a Windows user would hit.

**Lenses stay ephemeral** (Jeff, then Claude, 2026-11-20). A stored old lens would pull
the present toward staying true to it. A past lens can be rebuilt from the plans at that
commit, so it's derived, never stored. What a lens's notes worked out, if it matters, is
restated into a plan on purpose. Comparing then and now shows which reasons have
expired; a decision whose reason belonged to its moment ("a hack") is what that
comparison finds, and a revisit is what follows.

The lines that set the limits, kept because they're Jeff's:
- "If this starts to feel too much like a tracking/todo-list tool, developers will
  revolt." A lens has no state: no open or done, no date, no owner.
- A lens is a proto-plan: gather, with a question, then a conversation, then a draft or
  a plan. Each step is thrown away once the next holds what mattered.
