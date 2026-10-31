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
