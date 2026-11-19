# Seed: one server per machine, started when needed

From the persimmon cart (2026-11-19). It would revisit batik's port decision: today each
`phoropter` serves one repository on its own port (4173, or the next free one), records
itself in `~/.cache/phoropter/serving/`, and `phoropter url` finds it. Several can run
at once, even for one repository, by design.

The friction: someone has to start the server, and it seems to want a terminal of its
own. Jeff: "I think it would be awkward to have every user have to have a terminal
dedicated to running the server." Approaches considered in the cart:
- `ui` starts a server that stops when the `ui` does. Simple, but an agent reading the
  pages loses them when the person quits.
- A detached server that stops itself when idle.
- **`phoropter url` makes sure one is running** (Claude's lean): a single "ensure" that
  both the person and the agent already use, and `ui` calls it.
- **One server per machine, for every repository** (Jeff: "I *think* this is the right
  call"), with each repository under its own path: `http://localhost:4173/{repo-name}/`.
- `ui` builds to a cache and reads from disk, with no server. That fixes the person's
  half and not the agent's.
- No change, better words.

To settle in the draft:
- Two repositories can share a name (every clone of a course repo does), so the path
  needs something to tell them apart.
- `--at` serves an old commit (mock-control at `90c65b9`), so a path names a repository
  *and* a commit.
- Phoropter's links are relative, so a site under a prefix should already work.
  `shelf/pages.md` has the same `/<repo>/` question, so check it once for both.
- It changes poplin's "in the background it reads nothing" and batik's server. It's a
  revisit, asked in batik's frame.
