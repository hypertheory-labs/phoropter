import { test } from 'node:test';
import assert from 'node:assert/strict';
import { parsePlan, unwrapCodeSpan } from '../src/plan.ts';

test('code spans unwrap the CommonMark way', () => {
  assert.equal(unwrapCodeSpan('`a`'), 'a');
  assert.equal(unwrapCodeSpan('`` a`b ``'), 'a`b');
  assert.equal(unwrapCodeSpan("`selector: 'x',`"), "selector: 'x',");
  assert.equal(unwrapCodeSpan('not code'), 'not code');
});

test('entries, anchors, and quotes', () => {
  const doc = parsePlan(`---
cart: test
base: abc1234
---

# A plan

Intro.

## Decision: Do the thing
- where: \`src/a.ts › b\`
- quote: \`const b = 1;\`
- where (new): \`src/c.ts\`

**Because (Jeff):** reasons.

- a list that is prose

## Guess: It works
**Disproved if:** it doesn't.
`);
  assert.deepEqual(doc.frontmatter, { cart: 'test', base: 'abc1234' });
  assert.equal(doc.title, 'A plan');
  assert.equal(doc.intro, 'Intro.');
  assert.equal(doc.entries.length, 2);
  const [d, g] = doc.entries;
  assert.equal(d.kind, 'Decision');
  assert.equal(d.slug, 'do-the-thing');
  assert.deepEqual(d.anchors, [
    { where: 'src/a.ts › b', isNew: false, quote: 'const b = 1;' },
    { where: 'src/c.ts', isNew: true },
  ]);
  assert.match(d.body, /a list that is prose/);
  assert.equal(g.kind, 'Guess');
  assert.deepEqual(g.anchors, []);
});
