# Annotating a draft

How a person marks up a plan draft without anyone having to diff anything. This
restates the annotating agreement that
[annotations.nvim](https://github.com/JeffryGonzalez/annotations.nvim) implements,
with what phoropter adds for drafts. phoropter-tui writes notes by this page, and an
editor plugin that does the same can share a notes file with it.

## The protocol

**Whoever wrote a file doesn't annotate it. Whoever annotates a file doesn't edit it.**
Neither party ever has to reconcile two versions of one document.

For a draft, the agent usually writes `drafts/plan.draft.md` and the person annotates
it. Annotations go in **`drafts/plan.draft.notes.md`**, beside it. The draft is never changed by
annotating, and a notes file is never annotated.

**A file stays editable until it has notes.** Once a notes file exists, the draft is
fixed, because the quotes in the notes would come loose if it moved. Deleting the notes
releases it. Converting the draft to a plan deletes both the draft and its notes; what
mattered from the notes is in the plan's reasons by then.

## Notes on a landed plan

A landed plan is never edited, but it can take notes: a person saying it matters to
something now. For `plans/2026-10-31-ramie.md`, they go in
`drafts/2026-10-31-ramie.notes.md`, not beside the plan, so that `plans/` holds only
testimony. Their `annotates:` line names the plan's path. A note quotes an entry's
heading line, or has no quote and is about the whole plan. A landed plan never changes,
so the quotes can't come loose.

A note may say the plan's ground has moved, or "weigh this in what we're deciding", or
just say something. The agent answers in kind: a revisit (a draft whose `revisits:`
names the plan; converting it deletes the notes it answered), a citation in the
current draft, or a note in reply. *(phoropter's tarlatan and organza)*

## The notes file

```markdown
# Notes on plan.draft.md
<!-- annotates: plan.draft.md sha256:13029311acd1 -->

## — jeff

Small and good. Go when ready.

## L14 — jeff
> ## Decision: The panel stores its corner in localStorage

Is localStorage available everywhere this runs?
```

- **The `annotates:` line** names the draft and the start of its sha256, the version
  the notes were written against. Phoropter compares it with the draft it serves and
  says when the draft has moved since. Hash the draft's text without its final newline,
  as annotations.nvim does. Phoropter accepts either form.
- **Each note is a `##` heading, an optional blockquote of the exact source text, then
  the comment.** The heading gives a location and an author: `## L14 — jeff`,
  `## L37-38 — jeff`. Coordinates are optional (`## — jeff`).
- **The quote is the only part that must be exact.** Quote the draft's markdown, not the
  rendered page. Quote exactly what the note is about, a phrase or a block. Elide a long
  quote with `…` on its own line: first line, ellipsis, last line.
- **A note about a whole entry quotes the entry's heading line**, as in the example.
  Phoropter shows a note after the entry whose source holds its quote.
- **A note with no quote is about the whole draft.** It comes first.
- **Notes are in the draft's document order.**

## Prompts and their answers

A draft asks for a specific kind of answer with a heading keyword, at any heading level
below the entry it belongs to. Only the first word is matched, case-insensitively. The
prompt runs until the next heading of the same or a higher level. Phoropter adds one
line to every prompt: what happens if nobody answers it, since silence means proceed.

```markdown
### Question: Should the corner be per app or global?
**If unanswered:** per app, keyed by origin.

### Select: Which corners count?
- all four
- only the bottom two

**If unanswered:** all four.

### Affirm: Ship behind a flag?
**If unanswered:** no.
```

A `Select`'s options are the first bulleted list in its block, and any number may be
chosen. An `Affirm` is yes or no. Whoever writes a prompt lists only specific, castable
options, never "other" or "it depends": an answer that isn't on the list arrives in the
person's own words, with its reason.

**An answer restates the prompt's heading at `##`, word for word**, so the notes stand
alone, and so phoropter can match it to its prompt by kind and exact text:

```markdown
## Question: Should the corner be per app or global?
**Answer:** Per app. Two apps on one machine shouldn't fight over it.

## Select: Which corners count?
- [x] all four
- [ ] only the bottom two

**Note:** the top two are where the browser's own UI lives.

## Affirm: Ship behind a flag?
**Answer:** No.
```

- Any answer may carry a `**Note:**` with more to say. After a `Select`'s list, leave a
  blank line first. Directly under the list, markdown makes the note part of the last
  option, and phoropter's page shows it there. The agreement's own example doesn't have
  the blank line; this is the one place this page departs from it.
- **A prompt may be skipped on purpose**, so "considered and passed" stays different
  from "never got to it":

  ```markdown
  ## Question: Should the corner be per app or global?
  **Skipped.**
  ```

- Answering again replaces the earlier answer. The notes are the annotator's own file.

## What this doesn't constrain

A draft with prompts can still take notes anywhere, and a draft with none is annotated
the same way. A reader that doesn't recognize a heading's keyword treats it as an
ordinary note and keeps it.
