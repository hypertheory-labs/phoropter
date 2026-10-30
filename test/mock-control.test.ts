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
  const pumice = site.plans.find((p) => p.cart === 'pumice')!;
  assert.equal(pumice.before?.short, 'da023f1');

  const statuses = (slug: string) =>
    pumice.entries.find((e) => e.slug === slug)!.resolved[0].views.map((v) => `${v.role} ${v.status}`);

  // The selector quote matches at the base and not at the generated commit,
  // because pumice renamed it; the README heading moved for the same reason.
  assert.deepEqual(statuses('selectors-use-the-library-s-prefix-mock-panel'), ['base found', 'generated not-found']);
  assert.deepEqual(statuses('the-readme-is-a-setup-guide-for-an-angular-app'), ['base found', 'generated not-found']);
  assert.deepEqual(statuses('publish-to-npm-later-and-stay-at-0-0-1'), ['base found', 'generated found']);

  // Each found quote matched exactly once (the plan's second guess).
  for (const e of pumice.entries) for (const a of e.resolved) for (const v of a.views) if (v.status === 'found') assert.equal(v.matches, 1);

  assert.ok(site.places.some((p) => p.href === 'symbols/projects/mock-control/src/lib/panel/mock-panel.ts/MockPanel/'));
});
