# Seed: `phoropter affected`

Jeff, 2026-11-19 (marigold): "`phoropter affected --file <path>`, and it could create a
lens for every plan this file has been involved in", like `nx affected`, and in the
spirit of "LSP as a last resort".

It's file view version 1 (`shelf/file-pages.md`) as a command, and it's reachable: file
anchors match by path, symbol anchors through the index, which maps each symbol to its
file. It prints a lens; redirecting it into `drafts/lenses/` saves one, so phoropter
still writes only when asked.

The bigger half is nx's own direction, from a change rather than a file: `--since main`
lists every entry whose ground the diff touches. That's the README drift check
(`shelf/documentation.md`), an agent's habit before editing, and poplin's "what's
already at a draft's places" for any diff. Catches: indexing HEAD costs a run unless
cached, and a symbol that moved files resolves where it is now. The name is a draft
question (`why <file>`?).
