// batik: indexes are facts about commits, stored apart from pages. Against a
// clone of mock-control, so committing an index never touches the real one.
// Skipped when mock-control isn't checked out beside this repo.

import './cache.ts';
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { existsSync, mkdtempSync, readFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { Repo } from '../src/git.ts';
import { buildSite } from '../src/site.ts';
import { defaultIndexer, Store, type Indexer } from '../src/store.ts';

const source = new URL('../../mock-control', import.meta.url).pathname;
const skip = !existsSync(source) && 'no ../mock-control';
const PKG = 'scip-typescript @hypertheory-labs/mock-control';

function clone(): Repo {
  const dir = mkdtempSync(join(tmpdir(), 'phoropter-store-'));
  execFileSync('git', ['clone', '-q', source, dir]);
  return new Repo(dir);
}

function counting(): { indexer: Indexer; calls: string[] } {
  const calls: string[] = [];
  return { calls, indexer: (repo, sha, scheme, packages) => (calls.push(sha.slice(0, 7)), defaultIndexer(repo, sha, scheme, packages)) };
}

const symbolViews = (repo: Repo, store: Store) => {
  const site = buildSite(repo, 'f02528f', store);
  const pumice = site.plans.find((p) => p.name === 'pumice')!;
  const entry = pumice.entries.find((e) => e.slug === 'selectors-use-the-library-s-prefix-mock-panel')!;
  return entry.resolved[0].views.map((v) => `${v.role} ${v.status}`);
};

test('a second build reads the cache and indexes nothing', { skip }, () => {
  const repo = clone();
  const cache = mkdtempSync(join(tmpdir(), 'phoropter-cache-'));
  const first = counting();
  const views = symbolViews(repo, new Store(repo, { cache, indexer: first.indexer }));
  assert.ok(first.calls.length > 0);
  const second = counting();
  assert.deepEqual(symbolViews(repo, new Store(repo, { cache, indexer: second.indexer })), views);
  assert.deepEqual(second.calls, []);
});

test('phoropter index commits what plans need, and a build then needs no indexer', { skip }, () => {
  const repo = clone();
  const store = new Store(repo, { cache: mkdtempSync(join(tmpdir(), 'phoropter-cache-')) });
  const views = symbolViews(repo, store);
  const base = repo.resolve('da023f1')!;
  const path = store.commit(base, PKG)!;
  assert.equal(path, join(repo.dir, '.phoropter/index', base, '@hypertheory-labs+mock-control.json'));
  assert.equal(store.commit(base, PKG), undefined); // already there

  // The file is only what phoropter reads, sorted, without its own commit.
  const stored = JSON.parse(readFileSync(path, 'utf8'));
  assert.deepEqual(Object.keys(stored), ['indexer', 'version', 'scheme', 'package', 'definitions']);
  assert.equal(stored.scheme, 'scip-typescript');
  const keys = Object.keys(stored.definitions);
  assert.deepEqual(keys, [...keys].sort());
  assert.ok(!readFileSync(path, 'utf8').includes(base));

  // Every commit the anchors asked for committed, a fresh cache, and an
  // indexer that would fail: the build reads the store alone.
  for (const [sha, pkgs] of store.asked) for (const pkg of pkgs) store.commit(sha, pkg);
  const broken: Indexer = () => {
    throw new Error('scip-typescript is not installed');
  };
  const fresh = new Store(repo, { cache: mkdtempSync(join(tmpdir(), 'phoropter-cache-')), indexer: broken });
  assert.deepEqual(symbolViews(repo, fresh), views);
});

test('an indexer that fails leaves no-index on the view, not a failed build', { skip }, () => {
  const repo = clone();
  const broken: Indexer = () => {
    throw new Error('scip-typescript is not installed');
  };
  const store = new Store(repo, { cache: mkdtempSync(join(tmpdir(), 'phoropter-cache-')), indexer: broken });
  assert.deepEqual(symbolViews(repo, store), ['base no-index', 'as-built head no-index', 'generated no-index']);
});
