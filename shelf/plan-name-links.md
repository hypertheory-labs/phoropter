# Seed: a plan's name in prose links to the plan

From Jeff's note on tarlatan: when a reason says "nankeen said", could that be a link?
Plans refer to each other by name all the time, in prose, because the guide asks that
a plan be named with what it's about ("nankeen (landed plans aren't edited from the
TUI)"). Phoropter knows every plan's name, so the renderer could link a name it
recognizes in a reason or an intro, with `rel` saying so, the way an anchor's target
links an entry. Careful with: names that are ordinary words, names inside code spans,
and whether a name-in-prose link is a reference phoropter should list under the plan's
"pointed at by" (probably not: it's prose, not an anchor). Way future.
