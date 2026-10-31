// seersucker: indexers are programs in indexers/, chosen by the scheme an
// anchor's symbol starts with, and checked against indexers/README.md.

import './cache.ts';
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { lastName } from '../src/check.ts';
import { checkIndexer } from '../src/check.ts';
import { Repo } from '../src/git.ts';
import { indexerFor } from '../src/indexers.ts';
import { isScipSymbol, packageKey } from '../src/scip.ts';
import { Store } from '../src/store.ts';

const here = new Repo(new URL('..', import.meta.url).pathname);
const indexersDir = new URL('../indexers/', import.meta.url).pathname;
const noScipGo = spawnSync('sh', ['-c', 'command -v scip-go || test -x "${GOPATH:-$HOME/go}/bin/scip-go"']).status !== 0 && 'scip-go not installed';

test('any scheme is a symbol, and a › path never is', () => {
  assert.ok(isScipSymbol('scip-typescript npm @hypertheory-labs/mock-control . src/lib/panel/`mock-panel.ts`/MockPanel#'));
  assert.ok(isScipSymbol('scip-go gomod phoropter-tui . `phoropter-tui/internal/tui`/docScreen#annotate().'));
  assert.ok(isScipSymbol('roslyn nuget Contoso.Orders . Contoso/Orders/OrderService#Place().'));
  assert.ok(!isScipSymbol('README.md › Setup › 1. Install'));
  assert.ok(!isScipSymbol('src/render.ts'));
  assert.equal(packageKey('scip-go gomod phoropter-tui . `phoropter-tui/internal/tui`/form#'), 'scip-go phoropter-tui');
  assert.equal(lastName('scip-go gomod phoropter-tui . `phoropter-tui/internal/tui`/docScreen#annotate().'), 'annotate');
  assert.equal(lastName('roslyn nuget Contoso.Orders . Contoso/Orders/OrderService#`.ctor`().'), '.ctor');
  assert.equal(lastName('scip-typescript npm p . src/`mock-panel.ts`/MockPanel#'), 'MockPanel');
});

test('the first-party indexers are found by scheme', () => {
  assert.equal(indexerFor('scip-typescript')?.indexer, 'phoropter-index-typescript');
  assert.equal(indexerFor('nobody-has-this'), undefined);
});

test('a Go symbol in this repository resolves through the store', { skip: noScipGo }, () => {
  const store = new Store(here);
  const { definitions, failed } = store.lookup(here.resolve('HEAD')!, new Set(['scip-go phoropter-tui']));
  assert.equal(failed.size, 0);
  const annotate = definitions.get('scip-go gomod phoropter-tui . `phoropter-tui/internal/tui`/docScreen#annotate().');
  assert.equal(annotate?.file, 'tui/internal/tui/screens.go');
  assert.ok(annotate!.endLine > annotate!.startLine, 'a function spans its body, not just its name');
});

test('both first-party indexers pass --check', { skip: noScipGo }, () => {
  for (const name of ['typescript', 'go']) {
    const dir = indexersDir + name;
    const { command } = JSON.parse(spawnSync('cat', [`${dir}/indexer.json`], { encoding: 'utf8' }).stdout);
    const { ok, report } = checkIndexer({ command, cwd: dir }, here);
    assert.ok(ok, report.join('\n'));
  }
});

test('--check says what is wrong with an indexer that breaks the contract', () => {
  const liar = ['node', '-e', `
    if (process.argv[1] === '--describe') { console.log(JSON.stringify({ indexer: 'liar', version: '0', scheme: 'liar', extensions: ['.ts'] })); process.exit(0) }
    console.log(JSON.stringify({ indexer: 'liar', version: '0', scheme: 'liar', packages: { p: { definitions: {
      'liar npm p 1.0 src/Thing#': ['src/nowhere.ts', 1, 1],
      'liar npm q . src/Other#': ['src/store.ts', 1, 1],
    } } } }))`, '--'];
  const { ok, report } = checkIndexer({ command: liar }, here);
  assert.ok(!ok);
  const text = report.join('\n');
  assert.match(text, /version should be written as "\."/);
  assert.match(text, /package q, but it's listed under p/);
});

// Names an indexer makes up (`.ctor`) aren't looked for, and a few names a
// language spells differently (`op_Addition` is `operator +`) are notes. Wrong
// ranges still fail, because then nearly every name is missing.
test('--check reads made-up and differently spelled names, and still catches bad ranges', () => {
  const sha = here.resolve('HEAD')!;
  const text = here.show(sha, 'src/store.ts')!.split('\n');
  const at = (needle: string) => text.findIndex((l) => l.includes(needle)) + 1;
  const store = at('export class Store {');
  const fake = (defs: Record<string, [string, number, number]>) => ['node', '-e', `
    const out = { indexer: 'fake', version: '0', scheme: 'roslyn', extensions: ['.cs'] };
    if (process.argv[1] === '--describe') { console.log(JSON.stringify(out)); process.exit(0) }
    console.log(JSON.stringify({ ...out, packages: { p: { definitions: ${JSON.stringify(defs)} } } }))`, '--'];
  const names = ['lookup', 'commit', 'isCommitted', 'committedPath', 'readAt', 'read', 'write', 'repo', 'cache', 'asked', 'indexer', 'memory', 'constructor', 'cachedPath', 'Lookup', 'Found', 'Indexer', 'Stored', 'STORE', 'cacheDir'];
  const good: Record<string, [string, number, number]> = {};
  for (const n of names) good[`roslyn nuget p . Thing#${n}().`] = ['src/store.ts', 1, text.length];
  good['roslyn nuget p . Store#`.ctor`().'] = ['src/store.ts', store, store + 5];
  good['roslyn nuget p . Store#op_Addition().'] = ['src/store.ts', store, store];
  const passing = checkIndexer({ command: fake(good) }, here);
  assert.ok(passing.ok, passing.report.join('\n'));
  assert.match(passing.report.join('\n'), /1 of 21 name\(s\) aren't spelled in their ranges/);

  const wrong: Record<string, [string, number, number]> = {};
  for (const n of names) wrong[`roslyn nuget p . Thing#${n}().`] = ['src/store.ts', 1, 1];
  const failing = checkIndexer({ command: fake(wrong) }, here);
  assert.ok(!failing.ok);
});
