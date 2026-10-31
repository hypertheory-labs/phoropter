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
