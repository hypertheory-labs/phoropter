// What a reader of the pages can get without prose or class names (selvage).
// Built from phoropter's own plans at selvage's base, which is always here.

import './cache.ts';
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { Repo } from '../src/git.ts';
import { renderSite } from '../src/render.ts';
import { buildSite } from '../src/site.ts';

const site = buildSite(new Repo(new URL('..', import.meta.url).pathname), 'e8107f3');
const pages = renderSite(site, { markdown: '# Guide', version: 'test' });
const all = (re: RegExp, s: string) => [...s.matchAll(re)];

test('the home list marks up as-builts, revisits, and entry counts', () => {
  const home = pages.find((p) => p.href === '')!.html;
  assert.equal(all(/rel="as-built" itemprop="asBuilt"/g, home).length, site.plans.filter((p) => p.asBuilt).length);
  assert.equal(all(/rel="revisits" itemprop="revisits"/g, home).length, 1); // burlap revisits quince
  assert.equal(all(/itemprop="entryCount"/g, home).length, site.plans.length + site.places.length);
});

test('each match and each excerpt says its lines', () => {
  let views = 0;
  let matches = 0;
  for (const { html } of pages) {
    // Every code is directly preceded by the codeStart that numbers it.
    for (const [, start, dataStart] of all(/<meta itemprop="codeStart" content="(\d+)"><pre itemprop="code" data-start="(\d+)">/g, html)) assert.equal(start, dataStart);
    assert.equal(all(/itemprop="codeStart"/g, html).length, all(/itemprop="code"/g, html).length);

    // A view has one match item per match, and its lines are the marked ones.
    for (const [view] of all(/<figure class="view[\s\S]*?<\/figure>/g, html)) {
      views++;
      const n = Number(/itemprop="matches" content="(\d+)"/.exec(view)![1]);
      const items = all(/itemprop="match" itemscope hidden><meta itemprop="startLine" content="(\d+)"><meta itemprop="endLine" content="(\d+)">/g, view);
      assert.equal(items.length, n);
      // Numbered the way a reader would: codeStart, then one line per newline.
      const marked = new Set<number>();
      for (const [, start, code] of all(/itemprop="codeStart" content="(\d+)"><pre[^>]*><code[^>]*>([\s\S]*?)<\/code>/g, view)) {
        code.split('\n').forEach((line, i) => line.startsWith('<mark') && marked.add(Number(start) + i));
      }
      for (const [, s, e] of items) for (let l = Number(s); l <= Number(e); l++) assert.ok(marked.has(l) || view.includes('sameAs'), `line ${l}`);
      matches += n;
    }

    // The code's text is the code: no line numbers written into it (osier).
    assert.ok(!html.includes('class="ln"'));

    // `new` has one shape.
    assert.ok(!/<span itemprop="new">/.test(html));
  }
  assert.ok(views > 50 && matches > 20, `${views} views, ${matches} matches`);
});
