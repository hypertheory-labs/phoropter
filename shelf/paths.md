# Seed: resolve `›` paths to line ranges

From the TUI's brief (`tui/docs/for-phoropter.md`, section 1). Selvage left these on the
shelf, since resolving a path is more than markup. They stand together, and the second
makes the other two possible:

- **`where (new)` never resolves**, even at the as-built head where the work made it
  (`new-never-resolved`). Resolve a new anchor at the as-built head and the generated
  commit by its `›` path. Lanyard's "nothing to resolve" is true at the base only.
- **A `›` path resolves to nothing; only the quote does** (`where-path-unresolved`).
  Resolve heading paths (markdown) and key paths (JSON) to a line range.
- **A view whose quote isn't found has no code** (`view-no-code`). Fall back to the `›`
  range, the way a symbol falls back to its declaration.
