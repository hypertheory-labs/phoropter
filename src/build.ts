// node src/build.ts <repo> [commit] [--out site]
// Writes the site for one repo at one commit. The commit defaults to HEAD.

import { copyFileSync, mkdirSync, rmSync, writeFileSync } from 'node:fs';
import { dirname, join, resolve } from 'node:path';
import { parseArgs } from 'node:util';
import { Repo } from './git.ts';
import { readGuide } from './guide.ts';
import { renderSite } from './render.ts';
import { buildSite, type Site } from './site.ts';
import { Store } from './store.ts';

export function build(repoDir: string, ref: string, out: string, store?: Store): { pages: number; title: string; site: Site } {
  const repo = store?.repo ?? new Repo(resolve(repoDir));
  const site = buildSite(repo, ref, store ?? new Store(repo));
  const pages = renderSite(site, readGuide());
  rmSync(out, { recursive: true, force: true });
  for (const page of pages) {
    const file = join(out, page.href, 'index.html');
    mkdirSync(dirname(file), { recursive: true });
    writeFileSync(file, page.html);
  }
  copyFileSync(new URL('./style.css', import.meta.url), join(out, 'style.css'));
  return { pages: pages.length, title: `${site.repoName} at ${site.generated.short}`, site };
}

export function cli(argv: string[]): { repo: string; ref: string; out: string; port: number } {
  const { values, positionals } = parseArgs({
    args: argv,
    allowPositionals: true,
    options: { out: { type: 'string', default: 'site' }, port: { type: 'string', default: '4173' } },
  });
  if (positionals.length < 1) {
    console.error('usage: <repo> [commit] [--out site] [--port 4173]');
    process.exit(2);
  }
  return { repo: positionals[0], ref: positionals[1] ?? 'HEAD', out: values.out!, port: Number(values.port) };
}

if (import.meta.main) {
  const args = cli(process.argv.slice(2));
  const started = Date.now();
  const { pages, title } = build(args.repo, args.ref, args.out);
  console.log(`${title}: ${pages} pages in ${args.out}/ (${Date.now() - started} ms)`);
}
