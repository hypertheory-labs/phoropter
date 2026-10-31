// SCIP symbols in plans are stored without the package version (lanyard), so
// both sides are compared with the version replaced by `.`.

// Indexing itself is done by programs in indexers/ (seersucker); this file
// is only the symbol grammar every indexer shares.

export type Definition = {
  symbol: string; // version-less, as stored
  file: string; // repo-relative
  startLine: number; // 1-based, inclusive
  endLine: number;
};

// A symbol in SCIP's grammar: scheme, manager, package, version, then
// descriptors ending in one of SCIP's suffixes. Any scheme, not only scip-*
// (seersucker), so `roslyn nuget …` is a symbol too. A `›` path is a file
// anchor, never a symbol.
export function isScipSymbol(where: string): boolean {
  return !where.includes('›') && /^[a-z][\w.-]* \S+ \S+ \S+ \S.*[#.):!/\]]$/.test(where);
}

export function schemeOf(symbol: string): string {
  return symbol.split(' ')[0];
}

// Which indexer's package a symbol belongs to: its scheme and package name.
export function packageKey(symbol: string): string {
  const [scheme, , pkg] = symbol.split(' ');
  return `${scheme} ${pkg}`;
}

export function withoutVersion(symbol: string): string {
  const parts = symbol.split(' ');
  if (parts.length < 5) return symbol;
  parts[3] = '.';
  return parts.join(' ');
}

export function packageName(symbol: string): string | undefined {
  return symbol.split(' ')[2];
}

// The definitions in one commit, keyed by version-less symbol. Indexes each
// directory whose package.json names a package the anchors ask for, using the
// tsconfig files beside it.
