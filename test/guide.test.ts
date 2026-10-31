// The guide is phoropter's, served on every site: built here from phoropter's
// own plans at twill's base, which is always in this repo.

import './cache.ts';
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readGuide } from '../src/guide.ts';
import { Repo } from '../src/git.ts';
import { renderSite } from '../src/render.ts';
import { buildSite } from '../src/site.ts';

test('every page links the guide, which says which phoropter it came from', () => {
  const site = buildSite(new Repo(new URL('..', import.meta.url).pathname), '9bdb261');
  const pages = renderSite(site, { markdown: '# Working with plans\n\nSee the [profile](../profile/).', version: 'commit abc1234' });

  const guide = pages.find((p) => p.href === 'guide/')!;
  assert.equal(guide.title, 'Working with plans');
  assert.match(guide.html, /From phoropter, commit abc1234\./);
  assert.match(guide.html, /href="\.\.\/profile\/"/);

  for (const page of pages) {
    const depth = page.href.split('/').filter(Boolean).length;
    assert.ok(page.html.includes(`<link rel="help" href="${'../'.repeat(depth)}guide/">`), page.href);
    assert.ok(page.html.includes(`<a class="home-link" href="${'../'.repeat(depth) || './'}">home</a>`), page.href); // tassel
  }
});

test('the guide is read from phoropter itself', () => {
  const guide = readGuide();
  assert.match(guide.markdown, /^# Working with plans/);
  assert.match(guide.version, /^(commit [0-9a-f]{7,}|version \S+)/);
});
