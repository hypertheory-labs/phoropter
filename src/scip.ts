// SCIP symbols in plans are stored without the package version (lanyard), so
// both sides are compared with the version replaced by `.`.

import { execFileSync } from 'node:child_process';
import { readFileSync, existsSync } from 'node:fs';
import { createRequire } from 'node:module';
import { basename, dirname, join } from 'node:path';
import type { Repo } from './git.ts';
import { removeDir } from './git.ts';

const require = createRequire(import.meta.url);

// The indexer, named in every stored index (batik).
export const INDEXER = { name: 'scip-typescript', version: '0.4.0' };

// scip-typescript ships its generated protobuf bindings; it has no public API
// for reading an index, so this reaches into dist. Pinned to 0.4.0. Loaded
// only when something has to be indexed, so a build whose indexes are all
// stored doesn't need it installed.
function scipTypescript(): { scip: any; main: string } {
  const { scip } = require('@sourcegraph/scip-typescript/dist/src/scip.js');
  const main = join(dirname(require.resolve('@sourcegraph/scip-typescript/package.json')), 'dist/src/main.js');
  return { scip, main };
}

export type Definition = {
  symbol: string; // as the index has it, with the version
  file: string; // repo-relative
  startLine: number; // 1-based, inclusive
  endLine: number;
};

export function isScipSymbol(where: string): boolean {
  return /^scip-[\w-]+ \S+ \S+ \S+ /.test(where);
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
export function indexCommit(repo: Repo, sha: string, packages: Set<string>): Map<string, Definition> {
  const definitions = new Map<string, Definition>();
  const packageDirs = repo
    .files(sha)
    .filter((f) => f.endsWith('package.json') && !f.includes('node_modules/'))
    .filter((f) => {
      try {
        return packages.has(JSON.parse(repo.show(sha, f) ?? '{}').name);
      } catch {
        return false;
      }
    })
    .map((f) => dirname(f));
  if (packageDirs.length === 0) return definitions;

  const { scip, main: indexer } = scipTypescript();
  const all = repo.files(sha);
  const tree = repo.checkout(sha);
  try {
    for (const pkgDir of packageDirs) {
      const cwd = join(tree, pkgDir);
      // A tsconfig.json if there is one; otherwise every tsconfig*.json beside
      // the package.json (mock-control has only tsconfig.lib/spec/prod).
      const tsconfigs = existsSync(join(cwd, 'tsconfig.json'))
        ? ['tsconfig.json']
        : all.filter((f) => dirname(f) === pkgDir && /^tsconfig.*\.json$/.test(basename(f))).map((f) => basename(f));
      const output = join(tree, `.phoropter-${definitions.size}.scip`);
      execFileSync(process.execPath, [indexer, 'index', '--no-progress-bar', '--output', output, ...tsconfigs], { cwd, stdio: 'ignore' });
      const index = scip.Index.deserializeBinary(readFileSync(output));
      for (const doc of index.documents) {
        for (const occ of doc.occurrences) {
          if (!(occ.symbol_roles & scip.SymbolRole.Definition) || occ.symbol.startsWith('local ')) continue;
          const key = withoutVersion(occ.symbol);
          if (definitions.has(key)) continue;
          const range: number[] = occ.enclosing_range.length ? occ.enclosing_range : occ.range;
          definitions.set(key, {
            symbol: occ.symbol,
            file: pkgDir === '.' ? doc.relative_path : `${pkgDir}/${doc.relative_path}`,
            startLine: range[0] + 1,
            endLine: (range.length === 4 ? range[2] : range[0]) + 1,
          });
        }
      }
    }
  } finally {
    removeDir(tree);
  }
  return definitions;
}
