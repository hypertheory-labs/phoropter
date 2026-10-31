---
base: f02528f
---

# Let the panel remember which corner it was in

A test draft, to see how a second reader sees a draft page.

## Decision: The panel stores its corner in localStorage
- where: `projects/mock-control/src/lib/panel/mock-panel.ts`
- quote: `selector: 'mock-panel',`

**Because (Claude, not asked):** people move it out of the way and it comes back.

### Question: Should the corner be per app or global?
**If unanswered:** per app, keyed by origin.

## Decline: A drag handle
**Because (Claude, not asked):** four corners are enough.

### Select: Which corners count?
- all four
- only the bottom two

**If unanswered:** all four.
