# Working with plans

A plan is a short markdown file that records what you were thinking before a piece of
work: what you decided, what you turned down, and what you guessed, each with its reason
and where the reason came from. The code shows what was done. The plan shows why, in a
form you can argue with later.

This guide is the format, in the present tense. Each rule names the plan that decided it,
so the reasoning is there when you want it. Phoropter, the tool serving this page, reads
plans written this way and links them to the code they're about.

## An example

From mock-control, a plan about getting a library ready to publish:

```markdown
## Decision: Selectors use the library's prefix, `mock-panel`
- where: `` scip-typescript npm @hypertheory-labs/mock-control . src/lib/panel/`mock-panel.ts`/MockPanel# ``
- quote: `selector: 'app-mock-panel',`

**Because (Jeff):** a rename is cheapest before the first publish. Breaking the class
apps is fine.
```

A heading says what kind of entry it is. The list says what code it's about, with a
quote of the exact text as it was. The bold lead says where the reason came from. The
quote no longer matches today, because this plan's own work renamed it, and phoropter
shows both versions side by side.

## 1. Write the plan

Name it `plans/YYYY-MM-DD-<name>.md`. The date is the day you say "we're implementing
this," and it's written only in the filename, so it can't disagree with a second copy.
The name is any unique word that means nothing, so it can't turn into a category.
*(Decided in lanyard, aide's first plan, which defined the format.)*

The frontmatter has `base:`, the commit you're deciding against. Then a `# Title`, and a
short opening paragraph saying what the work is and why now, before any entries:

```markdown
---
base: e92038c
---

# Get the library ready to publish

The class apps have used mock-control from inside the workspace. Before anyone installs
it from npm, its names and setup should be the ones we'd keep.

## Decision: …
```

### Entries

Three kinds, each a `##` heading:

- `## Decision: <title>`, then `**Because (<source>):**` and the reason in prose.
- `## Decline: <title>`, the same, for something you considered and didn't do. Only
  for alternatives someone actually raised. Never invent strawmen to look thorough.
- `## Guess: <title>`, then `**Disproved if:**` and what would show it wrong.

Reasons stay prose. A reason squeezed into a field gets shorter and worse.
*(lanyard)*

### Sources

Every reason says where it came from, in the parentheses: a person's name, `measured`,
or `Claude, not asked` (or whichever agent). Combine them when that's the truth: `Jeff
and Claude`, `measured, then Claude, not asked`. An agent told to do something won't ask
why, so the record has to show when nobody said. A reason the agent guessed stays
labeled that way until someone confirms it. *(lanyard)*

When a draft's open question went unanswered and its fallback became the decision, say
so: `Claude, unanswered, so its fallback`. *(phoropter: first used in the plans for links between plans, sorrel, and for source and date, marl)*

### Anchors

An entry can say what it's about, as a list under its heading:

- `- where:` a file path, optionally narrowed with `›`: a heading path in markdown
  (`README.md › Setup › 1. Install`), or a key path in JSON (`package.json › version`).
  Or a symbol, in SCIP's grammar, with the package version written as `.` so a
  release doesn't change it: `phoropter symbols <file>` lists a file's symbols ready to
  paste. A symbol's first word picks the indexer that finds it; `indexers/README.md` in
  phoropter's repository says how indexers work and how to write one. Or another
  plan's entry: `plans/<file>.md › <entry title>`.
- `- quote:` exact text from the file at the base, so a reader can tell which lines you
  meant. It's searched for only in the anchor's own file.
- `- where (new):` something the work will create. No quote, since it doesn't exist
  yet.

Values go in code spans. When a value contains a backtick, use a double-backtick span:
`` `a `b` c` `` breaks; ``` `` a `b` c `` ``` doesn't. *(lanyard; the double-backtick rule is phoropter's, after awl, aide's plan on when
as-builts freeze, wrote one wrong)*

Anchor at the code as it is when you decide. The plan's own work will move it, and
that isn't drift: drift is measured from where the work landed. *(lanyard)*

Keep a plan small enough for one person to review in one sitting. *(lanyard, where it's
still a guess)*

## 2. Commit it before the work

That's how git shows the plan came first. From then on it's never edited. You don't
revise what you believed; you write a new plan that links the old one. *(lanyard)*

## 3. Do the work, and depart from the plan when the code disagrees

Don't force a plan through code that doesn't share its understanding, and don't quietly
do something else. Depart, and write it down in the as-built. Whether the departure was
right is a later conversation. *(lanyard)*

## 4. Write the as-built

`<plan>.as-built.md`, beside the plan. Its frontmatter has `head:`, the commit where the
work landed. Its anchors point at that commit, because they're about code the work made.
*(aide's grommet, the second version of as-builts)*

Three kinds of entry:

- `## Changed: <title>`: the work departed from the plan. `**Because (<source>):**`
- `## Chose: <title>`: the plan left something open, and you decided it.
  `**Because (<source>):**`
- `## Answered: <the guess's own title>`: one of the plan's guesses, answered against its
  `Disproved if`. `**Answer (<source>):**`, then one of `held`, `disproved`, `partly
  disproved`, or `not tested`, then how much evidence that rests on.

Keep departures and choices apart. A reviewer wants the departures first, and `Chose:`
is the comfortable label. *(grommet; reading the verdict as structure is phoropter's burlap)*

Anything else worth knowing, like a finding that isn't an answer to a guess, goes in
the as-built's opening prose, not in an entry. *(phoropter: burlap's as-built, which filed a finding as an answer first)*

An as-built can say nothing changed. Nothing requires one, and nothing flags a missing
one. *(lanyard)*

## 5. It freezes

An as-built is frozen once someone else could have read it, because from then on
someone may have anchored to it, and it mustn't change underneath them. Before that, fix
it freely. When that is depends on how your team shares work: a push, a pull request, a
merge. If you're unsure, treat it as frozen once it's merged into your main line.
Anything learned after that belongs in a new plan or a revisit. Phoropter can't tell
when this happened and doesn't check; it's advice with a reason, not a rule it enforces.
*(aide's awl said "committed"; phoropter's twill, the plan that brought this guide
here, made it "once someone else could have read it")*

## 6. When an earlier plan's ground is wrong, revisit it

If the work runs into something an earlier plan decided or assumed, don't decide it
inside the current plan; that re-decides the old plan in the wrong frame. Stop, and say
so in the current as-built: `## Changed:` "stopped here; revisiting …". Then write a
revisit: a new plan with `revisits: plans/<file>.md` in its frontmatter, naming exactly
one plan. It asks the question in that plan's frame, and every reason says what moved:
the world, the code, or your understanding. The outcome can be "it still holds."
Revisiting your own plan is the same move. *(aide's bodkin, on revisits; phoropter's burlap is one, reversing quince's decision
not to read verdicts)*

Sometimes the earlier plan never wrote down what it assumed. Then the revisit anchors
where the assumption shows, in the code or the docs, and says it wasn't in the plan.
That records a gap in the old plan, which is worth knowing too. *(mock-control's tallow,
the first revisit, about MSW 3: "MSW 2" was a line in the README, never an entry in the
plan it revisited)*

To answer a guess from an earlier plan, restate it as a `## Guess:` in your plan,
anchored at the original, and answer it in your as-built. *(mock-control's vellum, following its README in a fresh app, answered pumice's
`@source` guess this way)*

## Drafts (optional)

Most plans are never drafts: they go straight to `plans/`. A draft is for when the
agent's confidence or the shared understanding is thin, or when the pair has agreed,
for a session, that the person sees every plan first, not to judge it but to follow
it. Who decides that is the pair's to work out. It's never an approval gate: a draft
serves the agent's confidence and the person's understanding, not permission.
*(phoropter's loden, which served the first draft)*

Work in progress lives in `drafts/`, at the repository's root, and `plans/` holds
testimony and nothing else. A repository has at most one draft, `drafts/plan.draft.md`.
One fixed name means anyone can see whether a plan is being drafted, and drafts can't
pile up into a backlog. *(phoropter's tarlatan, which moved it from the root)* Each branch or worktree has its own working tree, so parallel work isn't
blocked. A draft that wants to be two plans gets smaller; the rest waits for the next
one. Ignoring `drafts/` in git is a suggestion: on a branch, committing it lets a
mentor or senior read a draft before the code exists. On main, what lands is plans.

Annotations go in `drafts/plan.draft.notes.md` beside it, in the
[annotating agreement](https://github.com/JeffryGonzalez/annotations.nvim)'s format.
An open question is one of that agreement's prompts, at any heading level below the
entry it belongs to, with what happens if nobody answers:

```markdown
### Question: Should phoropter also look anywhere else for a draft?
**If unanswered:** no. One name, and nothing else to configure.
```

`Select` lists its options as the first bulleted list; `Affirm` is yes or no. A draft
can have no questions at all: "look, then tell me to go."

When the pair is confident, the draft converts, whether or not every prompt is
answered, since silence means proceed. Converting writes `plans/YYYY-MM-DD-<name>.md`
(the date is today, the day you say "we're implementing this"), folds each answer into
its entry's reason, labels a fallback that became the decision (`Claude, unanswered, so
its fallback`), drops the prompts, and deletes the draft and its notes. The notes
aren't kept: they're mostly the questions that lined up someone's understanding, and
what mattered from them is in the reasons now.

Phoropter's dev server shows the draft at `draft/`, re-read on every request, with its
anchors resolved, its prompts and their fallbacks and answers, and each annotation
after the entry it quotes. It says it's the working tree, not a commit. A built site
never contains a draft.

### Notes on a landed plan

A landed plan is never edited, but it can take notes: a person saying the plan matters
to something now. For `plans/2026-10-31-ramie.md`, the notes are
`drafts/2026-10-31-ramie.notes.md`, in the same format as a draft's, and their
`annotates:` line names the plan. A note quotes an entry's heading line to be about that
entry, or has no quote to be about the whole plan. The agreement puts notes beside their
file. These live in `drafts/` instead, so that `plans/` holds only testimony. A landed
plan never changes, so the quotes can't come loose.

A note may say the plan's ground has moved, or "weigh this in what we're deciding now",
or just say something. Answer it in kind:
- **A revisit,** when the ground has moved: a draft whose `revisits:` names the plan,
  and whose opening says what moved, citing the notes. Converting it deletes the notes
  it answered.
- **A citation,** when the plan bears on the work at hand: the current draft anchors the
  entry the note is about, and its reason weighs it.
- **A note in reply,** when the agent thinks the plan holds, or there's nothing to do.

Notes stay until the person who wrote them is satisfied, and nobody holds a veto: the
person can always ask for a revisit.

On a branch, committed notes are how a teammate says "look at my branch, here's what
I'm thinking about this old plan." Some teams keep a long-lived branch for this kind of
work. That's a way of working, not a rule. Phoropter's dev server shows a plan's notes
on its page, after the entries they quote, and counts them on the home page. A built
site never shows them. *(phoropter's tarlatan and organza)*

## Older plans

Plans are read as written. Plans from before a rule here existed may use earlier forms,
like an as-built that uses `Changed:` for everything, or a `cart:` line. Phoropter
shows them as they are and doesn't rewrite or flag them.

## Squash merges and rebases

Plans record commits: `base:` and `head:`. A squash merge or a rebase replaces the
work's commits with new ones, and the as-built was written before that happened, so its
`head:` names a commit that isn't on your main line. Once the branch is deleted, that
commit may be gone. Phoropter doesn't reconcile this today: when `head:` isn't a commit
it can find, it shows the text as written, and the as-built's anchors are shown only at
the generated commit. Plans read best on a history of meaningful commits that are
merged, not squashed. *(phoropter's twill; not yet seen in a real repository)*

## Reading the site

Every page says which repository and commit it was generated from. The home page lists
plans oldest first and the places they anchor; each place lists the plans that touch
it, newest first. The [profile](../profile/) defines every link relation and microdata
property, so an agent can read the same pages a person does.
