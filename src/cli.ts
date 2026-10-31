#!/usr/bin/env node
// phoropter [path] [--at <commit>] [--port <n>]
// phoropter build [path] [--at <commit>] [--out site]
// phoropter index [path] [--at <commit>]
// phoropter url [path]
// phoropter symbols <file>
// phoropter index --check <program> [path]
//
// One command, run inside a repository (batik). With no command it finds the
// repository's root, indexes what's missing, builds the site at HEAD, and
// serves it with the draft and notes in drafts/. "You run ph in the repo"
// (Jeff); everything else is secondary to that.

import { createHash } from 'node:crypto';
import { existsSync, mkdirSync, readdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { dirname, extname, join, relative, resolve } from 'node:path';
import { parseArgs } from 'node:util';
import { build } from './build.ts';
import { checkIndexer } from './check.ts';
import { removeDir, Repo } from './git.ts';
import { indexers, runIndexer } from './indexers.ts';
import { serve } from './serve.ts';
import { buildSite, type Site } from './site.ts';
import { cacheDir, Store } from './store.ts';

const COMMANDS = new Set(['build', 'index', 'url', 'symbols', 'help']);
const USAGE = `usage:
  phoropter [path] [--at <commit>] [--port <n>]   serve the repository you're in
  phoropter build [path] [--at <commit>] [--out site]   write the static site
  phoropter index [path] [--at <commit>]   commit the indexes plans need into .phoropter/index/
  phoropter url [path]   print the address phoropter is serving this repository at
  phoropter symbols <file> [--scheme <scheme>]   list the symbols defined in a file, ready for a plan's where:
  phoropter index --check <program or indexer dir> [path]   check an indexer against indexers/README.md`;

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
// repository, one per server, keyed by the repository's root and the server's
// process, removed when that server stops. One file per server, because two
// can serve one repository, and the first to stop mustn't erase the other.
function servingId(root: string): string {
  return createHash('sha256').update(root).digest('hex').slice(0, 16);
}

export function servingFile(root: string, pid = process.pid): string {
  return join(cacheDir(), 'serving', `${servingId(root)}.${pid}.json`);
}

// The newest live server for a repository. Files left by servers that died
// without tidying up are removed on the way.
export function servingUrl(root: string): string | undefined {
  const dir = join(cacheDir(), 'serving');
  if (!existsSync(dir)) return undefined;
  const live: { url: string; started: number }[] = [];
  for (const name of readdirSync(dir).filter((n) => n.startsWith(`${servingId(root)}.`) && n.endsWith('.json'))) {
    const { url, pid, started } = JSON.parse(readFileSync(join(dir, name), 'utf8')) as { url: string; pid: number; started?: number };
    try {
      process.kill(pid, 0);
      live.push({ url, started: started ?? 0 });
    } catch {
      rmSync(join(dir, name), { force: true });
    }
  }
  return live.sort((a, b) => b.started - a.started)[0]?.url;
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
    options: { at: { type: 'string' }, port: { type: 'string' }, out: { type: 'string' }, check: { type: 'string' }, scheme: { type: 'string' }, help: { type: 'boolean', short: 'h' } },
  });
  const command = positionals[0] && COMMANDS.has(positionals[0]) ? positionals.shift()! : 'serve';
  if (command === 'help' || values.help) {
    console.log(USAGE);
    return;
  }
  const target = command === 'symbols' ? (positionals.shift() ?? fail('symbols needs a file')) : undefined;
  const start = positionals[0] ?? (target ? dirname(resolve(target)) : process.cwd());
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

  if (command === 'symbols') {
    const rel = relative(root, resolve(target!));
    const ext = extname(rel);
    // Every indexer for the file's extension, or the one --scheme names. With
    // more than one, each scheme's symbols are listed under its own heading
    // (dimity): a plan author picks one, and the scheme picks the indexer.
    const all = indexers().filter((i) => i.extensions.includes(ext));
    const candidates = values.scheme ? all.filter((i) => i.scheme === values.scheme) : all;
    if (!all.length) fail(`no indexer here handles ${ext} files (see indexers/README.md)`);
    if (!candidates.length) fail(`no indexer for ${ext} files has scheme ${values.scheme}; there's ${all.map((i) => i.scheme).join(', ')}`);
    const sha = repo.resolve(at)!;
    const tree = repo.checkout(sha);
    const byScheme = new Map<string, [number, number, string][]>();
    try {
      for (const info of candidates) {
        const found: [number, number, string][] = [];
        const out = runIndexer(info, tree);
        for (const { definitions } of Object.values(out.packages)) {
          for (const [symbol, [f, s, e]] of Object.entries(definitions)) if (f === rel) found.push([s, e, symbol]);
        }
        if (found.length) byScheme.set(`${info.scheme} (${info.indexer})`, found);
      }
    } finally {
      removeDir(tree);
    }
    if (!byScheme.size) fail(`nothing defined in ${rel} at ${repo.short(sha)} (uncommitted changes aren't indexed)`);
    for (const [heading, found] of byScheme) {
      if (byScheme.size > 1) console.log(`${heading}:`);
      for (const [s, e, symbol] of found.sort((a, b) => a[0] - b[0] || a[2].localeCompare(b[2]))) console.log(`${byScheme.size > 1 ? '  ' : ''}${`${s}–${e}`.padEnd(10)} ${symbol}`);
    }
    return;
  }

  if (command === 'index' && values.check) {
    // A directory with an indexer.json (indexers/go, or the one you're
    // writing), or a command.
    const dir = resolve(values.check);
    const manifest = join(dir, 'indexer.json');
    const program = existsSync(manifest) ? { command: (JSON.parse(readFileSync(manifest, 'utf8')) as { command: string[] }).command, cwd: dir } : { command: values.check.split(/\s+/) };
    const { ok, report } = checkIndexer(program, repo);
    console.log(report.join('\n'));
    if (!ok) process.exit(1);
    return;
  }

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
  writeFileSync(file, JSON.stringify({ url: served.url, root, commit: served.site.generated.sha, pid: process.pid, started: Date.now() }) + '\n');
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
