// The plan's own measurements, against mock-control at 90c65b9. Skipped when
// mock-control isn't checked out beside this repo.

import { test } from 'node:test';
import assert from 'node:assert/strict';
import { existsSync } from 'node:fs';
import { Repo } from '../src/git.ts';
import { buildSite } from '../src/site.ts';

const dir = new URL('../../mock-control', import.meta.url).pathname;

test('pumice at 90c65b9', { skip: !existsSync(dir) && 'no ../mock-control' }, () => {
  const site = buildSite(new Repo(dir), '90c65b9');
  const pumice = site.plans.find((p) => p.name === 'pumice')!;
  assert.equal(pumice.before?.short, 'da023f1');

  const statuses = (slug: string) =>
    pumice.entries.find((e) => e.slug === slug)!.resolved[0].views.map((v) => `${v.role} ${v.status}`);

  // The selector quote matches at the base and not after, because pumice
  // renamed it: it's already gone at pumice's own as-built head, so the plan
  // did it. The README heading moved for the same reason.
  assert.deepEqual(statuses('selectors-use-the-library-s-prefix-mock-panel'), ['base found', 'as-built head not-found', 'generated not-found']);
  assert.deepEqual(statuses('the-readme-is-a-setup-guide-for-an-angular-app'), ['base found', 'as-built head not-found', 'generated not-found']);
  assert.deepEqual(statuses('publish-to-npm-later-and-stay-at-0-0-1'), ['base found', 'as-built head found', 'generated found']);

  // Each found quote matched exactly once (the plan's second guess).
  for (const e of pumice.entries) for (const a of e.resolved) for (const v of a.views) if (v.status === 'found') assert.equal(v.matches, 1);

  // A SCIP view shows the quote in context, or the declaration when the quote
  // is gone; the whole range is on the symbol page.
  const [base, head, generated] = pumice.entries.find((e) => e.slug === 'selectors-use-the-library-s-prefix-mock-panel')!.resolved[0].views;
  assert.deepEqual(base.hunks.map((h) => [h.start, h.lines.length, h.marked]), [[9, 5, [10]]]);
  assert.deepEqual(head.hunks.map((h) => [h.start, h.lines.length, h.marked]), [[9, 7, []]]);
  assert.equal(head.hunks[0].lines[1], "  selector: 'mock-panel',");
  // Nothing changed it since, so the generated view says so and shows no code.
  assert.equal(generated.sameAs, 'e92db76');
  assert.deepEqual(generated.hunks, []);

  const panel = site.places.find((p) => p.href === 'symbols/projects/mock-control/src/lib/panel/mock-panel.ts/MockPanel/')!;
  assert.equal(panel.code?.lines.length, 152 - 9 + 1);
});

test('vellum, ferrule, and tallow at f02528f', { skip: !existsSync(dir) && 'no ../mock-control' }, () => {
  const site = buildSite(new Repo(dir), 'f02528f');
  // Three of these plans share a cart line; each still gets its own page.
  // Three share a date too; the commits that added them give the order.
  assert.deepEqual(site.plans.map((p) => p.href), ['plans/pumice/', 'plans/vellum/', 'plans/ferrule/', 'plans/tallow/']);
  assert.deepEqual(site.plans.map((p) => `${p.date} ${p.added?.short}`), ['2026-10-29 90c65b9', '2026-10-30 5639bce', '2026-10-30 a19bf72', '2026-10-30 24f61b9']);

  // Every answer in grommet's kinds pairs with its guess, and back.
  const answers = site.plans.flatMap((p) => p.asBuilt?.entries ?? []).filter((e) => e.kind === 'Answered');
  assert.equal(answers.length, 4);
  for (const a of answers) {
    assert.equal(a.answers?.entry.kind, 'Guess');
    assert.equal(a.answers?.entry.title, a.title);
    assert.ok(a.answers?.entry.answeredBy.some((r) => r.entry === a));
  }
  // Each answer's opening word is its verdict (burlap).
  assert.deepEqual(answers.map((a) => a.verdict), ['held', 'disproved', 'held', 'held']);
  // Pumice's as-built predates grommet; its guess stays unanswered here.
  assert.deepEqual(site.plans[0].entries.filter((e) => e.kind === 'Guess').map((e) => e.answeredBy.length), [0]);

  // Tallow's Requirements quote: found at its base, gone at its head, still
  // gone at f02528f. Tallow's own work removed it.
  const requirements = tallowOf(site).entries.find((e) => e.slug === 'what-moved-is-our-understanding-not-the-world')!.resolved[0].views;
  assert.deepEqual(requirements.map((v) => `${v.role} ${v.short} ${v.status}${v.sameAs ? ` same as ${v.sameAs}` : ''}`), [
    'base 3c59d26 found',
    'as-built head 4d28b68 not-found',
    'generated f02528f not-found same as 4d28b68',
  ]);

  // Tallow revisits pumice, and pumice shows it without being edited.
  const [pumice, vellum, , tallow] = site.plans;
  assert.equal(tallow.revisits, pumice);
  assert.deepEqual(pumice.revisitedBy, [tallow]);

  // Anchors at another plan's entry resolve to the entry, both ways, and make
  // no file page for the plan's markdown.
  const targets = site.plans
    .flatMap((p) => (p.asBuilt ? [p, p.asBuilt] : [p]))
    .flatMap((d) => d.entries.flatMap((e) => e.resolved.filter((a) => a.entry).map((a) => `${d.file} -> ${a.entry!.doc.file}#${a.entry!.entry.slug}`)));
  assert.deepEqual(targets, [
    'plans/2026-10-30-vellum.md -> plans/2026-10-29-pumice.md#tailwind-finds-the-panel-s-classes-through-an-source-line-into-node-modules',
    'plans/2026-10-30-tallow.md -> plans/2026-10-29-pumice.md#selectors-use-the-library-s-prefix-mock-panel',
    'plans/2026-10-30-tallow.as-built.md -> plans/2026-10-30-vellum.as-built.md#pin-msw-2-in-the-install-step-rather-than-support-msw-3',
  ]);
  assert.equal(vellum.asBuilt!.entries[0].anchoredBy[0].plan, tallow);
  assert.ok(!site.places.some((p) => p.href.startsWith('files/plans/')));
});

function tallowOf(site: ReturnType<typeof buildSite>) {
  return site.plans.find((p) => p.name === 'tallow')!;
}

const aide = new URL('../../aide', import.meta.url).pathname;

test('aide at 248a2a5: anchors between plans, one with a quote', { skip: !existsSync(aide) && 'no ../aide' }, () => {
  const site = buildSite(new Repo(aide), '248a2a5');
  const anchors = site.plans.flatMap((p) => p.entries.flatMap((e) => e.resolved)).filter((a) => a.target.kind === 'plan');
  assert.equal(anchors.length, 7);
  assert.ok(anchors.every((a) => a.entry));

  // Grommet quotes one sentence of lanyard's entry: one view, inside the entry.
  const quoted = anchors.filter((a) => a.anchor.quote !== undefined);
  assert.equal(quoted.length, 1);
  assert.deepEqual(quoted[0].views.map((v) => [v.role, v.status, v.hunks[0].marked]), [['generated', 'found', [39]]]);
});
