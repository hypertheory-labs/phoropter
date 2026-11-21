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
- **Started in the wrong directory, it reads one repository and writes to another**
  (Jeff, 2026-11-21). The server ran in `../phoropter-containers`; the TUI started
  elsewhere, found no server for that repository, fell back to `localhost:4173`, and
  showed the other one's draft. Notes write to the working directory, so annotating
  failed (the draft's sha256 check refused, which is the right failure). Two things pick
  "which repository": the URL and the working directory. The blind fallback to 4173 is
  the singleton assumption; in a repository with no server, saying so ("nothing is
  serving this repository: run `phoropter` here, or give an address") beats guessing.
  Bears on `one-server.md` both ways. Jeff: reading from anywhere is right; say so
  with a banner, something like "not in this repository: read only", rather than
  letting the first write fail.
