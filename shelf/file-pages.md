# Seed: file pages a reader can place anchors from

From the TUI's brief (`tui/docs/for-phoropter.md`, section 2), for a file mode: the code
with its reasons beside it, like `git blame` with reasons instead of authors. Gimlet
declined the heat map and the LSP adapter for v0, "so nobody adds them helpfully"; Jeff
asked for this one.

1. **Ranges on a file page:** each anchoredBy item gets its view at the generated commit
   (status, `startLine`/`endLine`, the match).
2. **Symbol anchors on their file's page.** A symbol page names its file without
   linking one (`symbol-file-unlinked`).
3. **A declared way to find a place by path,** a URI template in the profile
   (`place-url-convention`).
4. **The file's git blob id at the generated commit,** so a reader overlays lines only
   when its local file matches.
5. Optional: a plain-text alternate of the file.

Anchors that can't be placed at the generated commit belong on the page as drift, not
dropped.

**Jeff's guard, 2026-11-01:** plans are the way into a codebase, so a file is only ever
reached from a plan: a "now" view (with its span lit), a place, or a lens. Then the stack
of plans touching the file is there to explore. No tree view and no file picker: "if we
add a tree-view file explorer, we've failed or given up." A file no plan leads to is a
finding: nobody wrote down why it's that way.

**Jeff, 2026-11-19 (persimmon cart): a reading mode in the TUI, maybe instead of an LSP.**
Show the code with the plans that touch it visible. Make a lens while browsing. Zoom out
from a plan's snippet to the code around it. Cycle with ctrl-up and ctrl-down through
every plan that touches these lines. The reason: "anything in VS Code (or other LSP
supporting editor) is assuming humans will want to edit the code." It keeps the guard:
you arrive from a plan's place and explore outward. Gimlet declined an LSP adapter for
v0, so an LSP would start as a revisit of that decline, and this may make it unneeded.
