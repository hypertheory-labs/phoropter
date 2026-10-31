#!/usr/bin/env node
// phoropter [path] [--at <commit>] [--port <n>]
// phoropter build [path] [--at <commit>] [--out site]
// phoropter index [path] [--at <commit>]
// phoropter url [path]
//
// One command, run inside a repository (batik). With no command it finds the
// repository's root, indexes what's missing, builds the site at HEAD, and
// serves it with the draft and notes in drafts/. "You run ph in the repo"
// (Jeff); everything else is secondary to that.

import { createHash } from 'node:crypto';
import { existsSync, mkdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { dirname, join, relative, resolve } from 'node:path';
import { parseArgs } from 'node:util';
import { build } from './build.ts';
import { Repo } from './git.ts';
import { serve } from './serve.ts';
import { buildSite, type Site } from './site.ts';
import { cacheDir, Store } from './store.ts';

const COMMANDS = new Set(['build', 'index', 'url', 'help']);
const USAGE = `usage:
  phoropter [path] [--at <commit>] [--port <n>]   serve the repository you're in
  phoropter build [path] [--at <commit>] [--out site]   write the static site
  phoropter index [path] [--at <commit>]   commit the indexes plans need into .phoropter/index/
  phoropter url [path]   print the address phoropter is serving this repository at`;

// The repository holding a path: its root, found by walking up to `.git`.
export function repoRoot(start: string): string | undefined {
  let dir = resolve(start);
  for (;;) {
    if (existsSync(join(dir, '.git'))) return dir;
    const parent = dirname(dir);
    if (parent === dir) return undefined;
    dir = parent;
  }
}

// Where `phoropter url` finds a running server: a small file outside the
// repository, keyed by its root, removed when the server stops.
export function servingFile(root: string): string {
  return join(cacheDir(), 'serving', `${createHash('sha256').update(root).digest('hex').slice(0, 16)}.json`);
}

export function servingUrl(root: string): string | undefined {
  const file = servingFile(root);
  if (!existsSync(file)) return undefined;
  const { url, pid } = JSON.parse(readFileSync(file, 'utf8')) as { url: string; pid: number };
  try {
    process.kill(pid, 0);
    return url;
  } catch {
    return undefined; // a server that stopped without tidying up
  }
}

// The plan commits whose indexes a build needed but found only in the cache.
export function uncommitted(site: Site, store: Store): [sha: string, pkg: string][] {
  const planCommits = new Set(site.plans.flatMap((p) => [p.before?.sha, p.asBuilt?.before?.sha]).filter((s): s is string => !!s));
  const out: [string, string][] = [];
  for (const [sha, pkgs] of store.asked) {
    if (!planCommits.has(sha)) continue;
    for (const pkg of pkgs) if (!store.isCommitted(sha, pkg)) out.push([sha, pkg]);
  }
  return out;
}

function fail(message: string): never {
  console.error(`phoropter: ${message}`);
  process.exit(1);
}

async function main(argv: string[]): Promise<void> {
  const { values, positionals } = parseArgs({
    args: argv,
    allowPositionals: true,
    options: { at: { type: 'string' }, port: { type: 'string' }, out: { type: 'string' }, help: { type: 'boolean', short: 'h' } },
  });
  const command = positionals[0] && COMMANDS.has(positionals[0]) ? positionals.shift()! : 'serve';
  if (command === 'help' || values.help) {
    console.log(USAGE);
    return;
  }
  const start = positionals[0] ?? process.cwd();
  const root = repoRoot(start) ?? fail(`it reads a git repository, and ${resolve(start)} isn't in one`);
  if (command === 'url') {
    const url = servingUrl(root);
    if (!url) fail(`it isn't serving ${root}; run phoropter there`);
    console.log(url);
    return;
  }
  const repo = new Repo(root);
  const at = values.at ?? 'HEAD';
  if (!repo.resolve(at)) fail(at === 'HEAD' ? `${root} has no commits yet; phoropter reads a repository at a commit` : `${at} isn't a commit in ${root}`);
  const store = new Store(repo);
  const short = (path: string) => relative(process.cwd(), path) || '.';

  if (command === 'index') {
    const site = buildSite(repo, at, store);
    const written = uncommitted(site, store)
      .map(([sha, pkg]) => store.commit(sha, pkg))
      .filter((p): p is string => !!p);
    if (!written.length) console.log('Nothing to write: every index the plans need is already in .phoropter/index/.');
    else console.log(`Wrote ${written.length} to commit:\n${written.map((p) => `  ${short(p)}`).join('\n')}`);
    return;
  }

  if (command === 'build') {
    const out = values.out ?? 'site';
    const started = Date.now();
    const { pages, title, site } = build(root, at, out, store);
    console.log(`${title}: ${pages} pages in ${short(resolve(out))}/ (${Date.now() - started} ms)`);
    remind(site, store);
    return;
  }

  // serve: the site goes in the cache, never into the repository.
  const out = values.out ?? join(cacheDir(), 'sites', createHash('sha256').update(root).digest('hex').slice(0, 16));
  const served = await serve({ repo: root, ref: at, out, port: Number(values.port ?? 4173), exactPort: values.port !== undefined, store });
  const file = servingFile(root);
  mkdirSync(dirname(file), { recursive: true });
  writeFileSync(file, JSON.stringify({ url: served.url, root, commit: served.site.generated.sha, pid: process.pid }) + '\n');
  const tidy = () => rmSync(file, { force: true });
  process.on('exit', tidy);
  for (const signal of ['SIGINT', 'SIGTERM'] as const) process.on(signal, () => process.exit(0));
  console.log(`${served.title}: ${served.pages} pages\nserving ${served.url}`);
  remind(served.site, store);
}

function remind(site: Site, store: Store): void {
  const missing = uncommitted(site, store);
  if (missing.length) console.log(`${missing.length} index${missing.length === 1 ? '' : 'es'} that plans need ${missing.length === 1 ? 'is' : 'are'} only in this machine's cache; phoropter index commits them.`);
}

if (import.meta.main) await main(process.argv.slice(2));
