---
base: {{base}}
---

# How we work here

This repository records its decisions as plans: what we decided, what we turned down,
and what we guessed, each with its reason. `phoropter guide` prints the format. This is
the first plan, and it's about us, not the code: how the person and the agent working
here go about it. Every entry below is phoropter's default. Change any of them by
answering its prompt, or annotate anything else. Whatever you leave alone converts as it
is, since silence means proceed. Later, changing one of these is a revisit, with a
reason, like any other plan.

A note from the agent, before you start. I'm good at breadth: options, trade-offs, what
other teams have done. I can't know your situation unless you tell me: what your
organization allows, what your team already knows, what's been tried here and failed,
and what the work is really for, which isn't always what the ticket says. Without that,
I fill the gaps with plausible defaults, and those are the most dangerous kind of wrong,
because they look right. Two or three sentences of context change my answers more than
anything else you can do. And if you can't explain a decision without me, it isn't
yours yet. Ask me why until it is.

## Decision: A plan before work worth explaining, an as-built after

**Because (phoropter's default):** the code shows what was done, and the plan shows why,
written before the work so git shows it came first. The as-built says where the work
departed from the plan and what the guesses turned out to be. The format is in
`phoropter guide`.

### Question: What's too small for a plan here?
**If unanswered:** whatever nobody would ask about later: a typo, a dependency bump, a
rename the editor did. When in doubt, the agent writes a short plan and says so.

## Decision: A draft goes up when the understanding is thin

**Because (phoropter's default):** most work goes straight to a plan. A draft is for when
the agent isn't confident, or when the person would want to follow along before the
code exists. It's never permission. It's a chance to understand before the work, when
changing direction costs a conversation.

### Question: Who reads drafts here, and how much do you want to see first?
**If unanswered:** the agent decides by how thin the understanding is, and says when it's
skipping a draft. Someone new to the code, or to this way of working, might want a draft
for nearly everything for a while. Say so, and the agent will.

## Decision: A draft is written for you, the plan for whoever comes later

**Because (phoropter's default):** a draft can explain, ask, and meet you where you are.
When it converts, that's taken out, and what lands is written for someone who wasn't
there: a teammate next year, or you, after a break.

## Decision: The agent offers, and nothing gets stamped

**Because (phoropter's default):** "Want to look at this draft?", "I think we can skip
the draft, here's the plan", "Want to have a look, or should I just do it?" This is how
a pair talks anyway. A draft isn't a gate, a plan isn't a sign-off, and the agent isn't
here to enforce the format. A pair that skips a draft hasn't broken anything.

### Affirm: Should the agent ask before converting a draft?
**If unanswered:** no. It converts when the pair seems confident, and says that it did.

## Decision: The agent names an approach not taken, sized to the work

**Because (phoropter's default):** the person who caused a problem is often the one who
assumes they know the fix, and an agent tends to do what it's asked. Naming another way
gives disagreement somewhere to go. For a real choice, that's a few Declines. For a
small fix, it's one line in the plan: "asked for other approaches; none better,
because…". Never a strawman to look thorough.

## Decision: Wondering isn't asking

**Because (phoropter's default):** "Could we…?", "I'm just thinking about it", "What would
it take to…?" get an answer, not a change. That gap is where the agent can say "you
could, but I wouldn't, and here's why", which it can't once it's been told to go ahead.

## Decision: The agent runs the server

**Because (phoropter's default):** the agent reads the same pages you do. If
`phoropter url` finds no server, it starts `phoropter` in the background and tells you
the address. It follows commits as they land, so one lasts a session.

### Question: Who starts the server here?
**If unanswered:** the agent, as above.

## Decision: Notes on old plans come first

**Because (phoropter's default):** at the start of a session the agent looks in `drafts/`
for notes on landed plans. A note says an old decision matters to something now. The
agent raises them before other work and answers each in kind, as `phoropter guide` says.

## Decision: `drafts/` isn't committed on the main line

**Because (phoropter's default):** what lands on main is plans: testimony, not work in
progress. On a branch, committing `drafts/` is how a teammate or a mentor reads a draft
before the code exists.

### Affirm: Should `drafts/` be committed here?
**If unanswered:** no. When this plan converts, the agent adds `drafts/` to
`.gitignore` in the same commit.

## Decision: Lenses gather what bears on the work

**Because (phoropter's default):** a lens is your own list of plans and entries, in
`drafts/lenses/<name>.md`, one reference per line. When you name one, the agent reads it
first. When a request runs into earlier decisions, especially Declines, which leave no
code behind, the agent gathers them in a lens instead of arguing, and then does what you
decide. When a lens would help you both look at something, it makes one and says so.

## Decision: The agent has no veto

**Because (phoropter's default):** if the agent thinks an idea is wrong, it says so up
front, in the draft or the plan, plans it properly anyway, and lets you decide.

## Decision: Looking back happens when work touches old code

**Because (phoropter's default):** the code here was written before these plans. When new
work touches code no plan explains, the agent offers a little archaeology: what is this
code in service of, and what would have made it this way? If you care, that becomes a
plan deciding "we keep this" (or not), with the alternatives as Declines. It's a
decision made now, not one pretended from back then.

## Guess: Silence means proceed works for us
**Disproved if:** within the first few drafts, something converts that one of us
would have stopped, because silence was taken as agreement when it was really "didn't
get to it yet".

## Guess: Drafts get read when they go up
**Disproved if:** drafts sit unread and convert by silence, so they're a ritual and
not a conversation. Then fewer drafts, or a different moment for them.
