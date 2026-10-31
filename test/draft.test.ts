// The one draft (loden): prompts with fallbacks, notes placed by their
// quotes, anchors resolved at commits, and nothing registered on the site.

import { test } from 'node:test';
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { mkdtempSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { parseNotes, readDraft, splitPrompts } from '../src/draft.ts';
import { Repo } from '../src/git.ts';
import { draftPage } from '../src/render.ts';
import { buildSite } from '../src/site.ts';

const draft = `---
base: f62365f
revisits: plans/2026-10-30-gimlet.md
---

# A draft

Why now.

## Decision: Serve it
- where: \`src/serve.ts\`
- quote: \`const { pages, title } = build(args.repo, args.ref, args.out);\`

**Because (Claude, not asked):** a reason
that runs two lines.

### Question: Which port?
Say anything.

**If unanswered:** 4173.

## Decline: Two drafts
**Because (Jeff):** a backlog.

### Select: Which of these?
- One
- Two

**If unanswered:** neither.
`;

const notes = `# Notes on plan.draft.md
<!-- annotates: plan.draft.md sha256:000000000000 -->

## — jeff

The whole thing reads well.

## L14-15 — jeff
> **Because (Claude, not asked):** a reason
> that runs two lines.

Why two?

## L20 — jeff
> ## Decline: Two drafts
> …
> **Because (Jeff):** a backlog.

Agreed.

## L3 — jeff
> text that was never there

Hm.

## Question: Which port?
**Answer:** 4174, 4173 is taken.
`;

test('prompts come out of the markdown, with options and fallbacks', () => {
  const { rest, prompts } = splitPrompts('Before.\n\n### Affirm: Ship it?\n**If unanswered:** yes,\nquietly.\n\n## Next\nAfter.');
  assert.equal(rest, 'Before.\n\n## Next\nAfter.');
  assert.deepEqual(prompts.map((p) => [p.kind, p.text, p.fallback]), [['Affirm', 'Ship it?', 'yes,\nquietly.']]);
  const select = splitPrompts('#### select: Pick\n- a\n- b\n\nNot an option.').prompts[0];
  assert.deepEqual([select.kind, select.options, select.body], ['Select', ['a', 'b'], 'Not an option.']);
});

test('notes: heading, quote, comment', () => {
  const [whole, quoted] = parseNotes(notes);
  assert.deepEqual(whole, { heading: '— jeff', quote: undefined, comment: 'The whole thing reads well.', found: true });
  assert.equal(quoted.quote, '**Because (Claude, not asked):** a reason\nthat runs two lines.');
  assert.equal(quoted.comment, 'Why two?');
});

test('a draft resolves against the site without joining it', () => {
  const site = buildSite(new Repo(new URL('..', import.meta.url).pathname), 'f62365f');
  const dir = mkdtempSync(join(tmpdir(), 'phoropter-draft-'));
  writeFileSync(join(dir, 'plan.draft.md'), draft);
  assert.equal(readDraft(site, dir)!.notes, undefined);
  writeFileSync(join(dir, 'plan.draft.notes.md'), notes);
  const d = readDraft(site, dir)!;

  assert.equal(d.plan.revisits?.name, 'gimlet');
  assert.equal(d.plan.revisits?.revisitedBy.length, 0);
  const [serve] = d.plan.entries;
  assert.deepEqual(serve.resolved[0].views.map((v) => `${v.role} ${v.status}`), ['generated found']);
  assert.ok(!serve.body.includes('Question'));
  assert.ok(!site.places.some((p) => p.refs.some((r) => r.doc.file === 'plan.draft.md')));

  const [question] = d.prompts.get('serve-it')!;
  assert.equal(question.fallback, '4173.');
  assert.equal(question.answer?.comment, '**Answer:** 4174, 4173 is taken.');
  assert.deepEqual(d.prompts.get('two-drafts')![0].options, ['One', 'Two']);

  const at = d.notes!.annotations.map((a) => [a.heading, a.found, a.at]);
  assert.deepEqual(at, [
    ['— jeff', true, undefined],
    ['L14-15 — jeff', true, 'serve-it'],
    ['L20 — jeff', true, 'two-drafts'],
    ['L3 — jeff', false, undefined],
  ]);
  assert.equal(d.notes!.annotates, '000000000000');
  assert.equal(d.notes!.current, false);

  // The hash annotations.nvim writes: the lines joined, without the final newline.
  const nvim = createHash('sha256').update(draft.replace(/\n$/, '')).digest('hex').slice(0, 12);
  writeFileSync(join(dir, 'plan.draft.notes.md'), notes.replace('000000000000', nvim));
  assert.equal(readDraft(site, dir)!.notes!.current, true);

  const html = draftPage(site, d).html;
  assert.match(html, /Draft, not committed: <code itemprop="file">plan.draft.md<\/code>/);
  assert.match(html, /an earlier version of the draft; quotes may have come loose/);
  assert.match(html, /quote not found in the draft/);
  assert.ok(html.indexOf('Why two?') < html.indexOf('Two drafts</span>'));
  assert.ok(html.indexOf('Agreed.') > html.indexOf('Two drafts</span>'));
});
