// phoropter-index-typescript: phoropter's indexer for TypeScript (seersucker).
// It runs scip-typescript on each npm package the anchors ask for, and prints
// the definitions as phoropter's index JSON. See indexers/README.md.
//
//   node index.ts --describe
//   node index.ts <dir> [package name...]
//
// A package is found by the `name` in its package.json, which is also the
// package field of its symbols: scip-typescript npm @scope/name . src/`file.ts`/Thing#

import { execFileSync } from 'node:child_process';
import { existsSync, mkdtempSync, readdirSync, readFileSync, rmSync } from 'node:fs';
import { createRequire } from 'node:module';
import { tmpdir } from 'node:os';
import { dirname, join, relative, sep } from 'node:path';

const VERSION = '0.1.0';
const SCIP_TYPESCRIPT = '0.4.0';
const require = createRequire(import.meta.url);

if (process.argv[2] === '--describe') {
  const describe: Record<string, unknown> = { indexer: 'phoropter-index-typescript', version: VERSION, scheme: 'scip-typescript', extensions: ['.ts', '.tsx', '.mts', '.cts'] };
  // What this needs to run, for phoropter doctor (grosgrain): scip-typescript
  // comes from npm install in phoropter's clone.
  try {
    require.resolve('@sourcegraph/scip-typescript/package.json');
  } catch {
    describe.needs = [{ what: `scip-typescript ${SCIP_TYPESCRIPT}`, install: `npm install, in ${dirname(dirname(import.meta.dirname))}` }];
  }
  console.log(JSON.stringify(describe));
  process.exit(0);
}
const dir = process.argv[2];
if (!dir) fail('usage: node index.ts <dir> [package name...]');

// scip-typescript ships its generated protobuf bindings; it has no public API
// for reading an index, so this reaches into dist. Pinned to 0.4.0.
const { scip } = require('@sourcegraph/scip-typescript/dist/src/scip.js');
const main = join(dirname(require.resolve('@sourcegraph/scip-typescript/package.json')), 'dist/src/main.js');

// The contract's paths use /; on Windows, Node's and scip-typescript's use \
// (cambric, found on the trial).
const slash = (p: string): string => p.split(sep).join('/');

// Every package.json outside node_modules, by its name.
function packages(root: string): Map<string, string> {
  const found = new Map<string, string>();
  const walk = (d: string) => {
    for (const e of readdirSync(d, { withFileTypes: true })) {
      if (e.isDirectory()) {
        if (e.name !== 'node_modules' && e.name !== '.git') walk(join(d, e.name));
      } else if (e.name === 'package.json') {
        try {
          const name = JSON.parse(readFileSync(join(d, e.name), 'utf8')).name;
          if (name && !found.has(name)) found.set(name, d);
        } catch {
          // not JSON, or no name: not a package
        }
      }
    }
  };
  walk(root);
  return found;
}

const all = packages(dir);
const asked = process.argv.length > 3 ? process.argv.slice(3) : [...all.keys()];
const out = { indexer: 'phoropter-index-typescript', version: `${VERSION} (scip-typescript ${SCIP_TYPESCRIPT})`, scheme: 'scip-typescript', packages: {} as Record<string, { definitions: Record<string, [string, number, number]> }>, missing: [] as string[] };
const scratch = mkdtempSync(join(tmpdir(), 'phoropter-index-typescript-'));
try {
  for (const name of asked) {
    const pkgDir = all.get(name);
    if (!pkgDir) {
      out.missing.push(name);
      continue;
    }
    // A tsconfig.json if there is one; otherwise every tsconfig*.json beside
    // the package.json (mock-control has only tsconfig.lib/spec/prod).
    const tsconfigs = existsSync(join(pkgDir, 'tsconfig.json'))
      ? ['tsconfig.json']
      : readdirSync(pkgDir).filter((f) => /^tsconfig.*\.json$/.test(f));
    const output = join(scratch, `${out.missing.length}-${Object.keys(out.packages).length}.scip`);
    execFileSync(process.execPath, [main, 'index', '--no-progress-bar', '--output', output, ...tsconfigs], { cwd: pkgDir, stdio: 'ignore' });
    const index = scip.Index.deserializeBinary(readFileSync(output));
    const prefix = slash(relative(dir, pkgDir));
    const definitions: Record<string, [string, number, number]> = {};
    for (const doc of index.documents) {
      for (const occ of doc.occurrences) {
        if (!(occ.symbol_roles & scip.SymbolRole.Definition) || occ.symbol.startsWith('local ')) continue;
        const parts = occ.symbol.split(' ');
        if (parts.length < 5 || parts[2] !== name) continue;
        parts[3] = '.';
        const key = parts.join(' ');
        if (key in definitions) continue;
        const r: number[] = occ.enclosing_range.length ? occ.enclosing_range : occ.range;
        definitions[key] = [prefix ? `${prefix}/${slash(doc.relative_path)}` : slash(doc.relative_path), r[0] + 1, (r.length === 4 ? r[2] : r[0]) + 1];
      }
    }
    out.packages[name] = { definitions };
  }
} catch (err) {
  fail(`scip-typescript failed: ${(err as Error).message.split('\n')[0]}`);
} finally {
  rmSync(scratch, { recursive: true, force: true });
}
console.log(JSON.stringify(out));

function fail(message: string): never {
  console.error(message);
  process.exit(1);
}
