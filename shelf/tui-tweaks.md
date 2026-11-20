# Various little observations about the feel/use of the tui 

- Adding notes, etc - don't seem to be able to move backwards with the arrow keys (or other mechanisms) - notice a typo five words earlier? backspace, fix, retype. 
  - ~~super stretch goal - make the notes modal, and add some vim motions.~~ No: opening the note in `$EDITOR` already covers that; try it next round. (Jeff, 2026-11-20)
- Super low priority, and I think we should have our "browse the code" story together first, but the possibility of having some light syntax highlighting would be good polish.
- **Open on the active lens, not the plan stack** (Jeff and Claude, 2026-11-20). A list
  of every plan is Obsidian's graph view: impressive, rarely useful. Real work starts
  from a place (a stack trace's file: `phoropter affected`) or a topic ("security", "API
  validation": the agent builds the lens by meaning). In an incident the order that
  helps is open guesses near the code, then Declines, then what changed recently. The
  full plan list is somewhere you go, not where you land. For the rewrite, if not
  sooner.
