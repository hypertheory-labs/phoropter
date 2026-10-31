// batik: the phoropter command's pieces that don't need a running server.

import './cache.ts';
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { mkdirSync, mkdtempSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';
import { repoRoot, servingFile, servingUrl } from '../src/cli.ts';

test('the repository is found by walking up to .git', () => {
  const root = mkdtempSync(join(tmpdir(), 'phoropter-root-'));
  mkdirSync(join(root, '.git'));
  mkdirSync(join(root, 'a', 'b'), { recursive: true });
  assert.equal(repoRoot(join(root, 'a', 'b')), root);
  assert.equal(repoRoot(mkdtempSync(join(tmpdir(), 'phoropter-nowhere-'))), undefined);
});

test('phoropter url finds a live server, and not one that stopped', () => {
  const root = mkdtempSync(join(tmpdir(), 'phoropter-served-'));
  assert.equal(servingUrl(root), undefined);
  const file = servingFile(root);
  mkdirSync(dirname(file), { recursive: true });
  writeFileSync(file, JSON.stringify({ url: 'http://localhost:4174/', root, commit: 'abc', pid: process.pid }));
  assert.equal(servingUrl(root), 'http://localhost:4174/');
  writeFileSync(file, JSON.stringify({ url: 'http://localhost:4174/', root, commit: 'abc', pid: 2 ** 22 + 7 }));
  assert.equal(servingUrl(root), undefined);
});
