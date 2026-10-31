// The index store (batik). An index is a fact about one commit: in this
// package, at this sha, these symbols are defined on these lines. It never
// changes, so it's kept rather than recomputed.
//
// - The indexes plans need (their base and as-built commits) are committed, in
//   `.phoropter/index/<sha>/<package>.json`, so a plan's symbol anchors outlive
//   the toolchain that could index them. Only `phoropter index` writes there;
//   a build never writes into the repository it reads.
// - Everything else, the generated commit's index above all, goes in a local
//   cache under $XDG_CACHE_HOME/phoropter (or ~/.cache/phoropter).
//
// A file holds only what phoropter reads, sorted, and not its own commit (the
// path has it), so an unchanged package gives byte-identical files across
// commits and git stores them once. It names its scheme and indexer, so two
// schemes naming the same package read as missing rather than misread
// (seersucker).
//
// Packages are keyed `<scheme> <package>`, as packageKey() gives them: the
// scheme picks the indexer that runs (src/indexers.ts).

import { createHash } from 'node:crypto';
import { existsSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { homedir } from 'node:os';
import { join } from 'node:path';
import { removeDir, type Repo } from './git.ts';
import { definitionsOf, indexerFor, runIndexer } from './indexers.ts';
import type { Definition } from './scip.ts';

export const STORE = '.phoropter/index';

type Stored = {
  indexer: string;
  version: string;
  scheme?: string; // absent in files written before seersucker, which were all scip-typescript
  package: string;
  definitions: Record<string, [file: string, startLine: number, endLine: number]>;
};

export function cacheDir(): string {
  return join(process.env.XDG_CACHE_HOME || join(homedir(), '.cache'), 'phoropter');
}

// A package name as a file name: `@scope/name` becomes `@scope+name`.
function fileOf(pkg: string): string {
  return `${pkg.replaceAll('/', '+')}.json`;
}

function split(key: string): [scheme: string, pkg: string] {
  const at = key.indexOf(' ');
  return [key.slice(0, at), key.slice(at + 1)];
}

export function serialize(found: { indexer: string; version: string; scheme: string }, pkg: string, defs: Map<string, Definition>): string {
  const definitions: Stored['definitions'] = {};
  for (const key of [...defs.keys()].sort()) {
    const d = defs.get(key)!;
    definitions[key] = [d.file, d.startLine, d.endLine];
  }
  const stored: Stored = { indexer: found.indexer, version: found.version, scheme: found.scheme, package: pkg, definitions };
  return JSON.stringify(stored, null, 1) + '\n';
}

function deserialize(text: string): { scheme: string; defs: Map<string, Definition> } {
  const stored = JSON.parse(text) as Stored;
  const defs = new Map<string, Definition>();
  for (const [key, [file, startLine, endLine]] of Object.entries(stored.definitions)) {
    defs.set(key, { symbol: key, file, startLine, endLine });
  }
  return { scheme: stored.scheme ?? 'scip-typescript', defs };
}

// What indexing one scheme's packages at one commit found: definitions by
// package. A package the indexer couldn't find is absent.
export type Found = { indexer: string; version: string; scheme: string; packages: Map<string, Map<string, Definition>> };
export type Indexer = (repo: Repo, sha: string, scheme: string, packages: string[]) => Found;

// The scheme's indexer, run on the commit checked out on its own.
export const defaultIndexer: Indexer = (repo, sha, scheme, packages) => {
  const info = indexerFor(scheme);
  if (!info) throw new Error(`no indexer for ${scheme} here (phoropter-index-${scheme} isn't on PATH)`);
  const tree = repo.checkout(sha);
  try {
    const out = runIndexer(info, tree, packages);
    const found = new Map<string, Map<string, Definition>>();
    for (const pkg of packages) {
      const defs = definitionsOf(out, pkg);
      if (defs) found.set(pkg, defs);
    }
    return { indexer: out.indexer, version: out.version, scheme, packages: found };
  } finally {
    removeDir(tree);
  }
};

// What resolving at one commit gets: the definitions found, and the packages
// (by key) that couldn't be indexed, whose anchors show `no-index`.
export type Lookup = { definitions: Map<string, Definition>; failed: Map<string, string> };

export class Store {
  readonly repo: Repo;
  readonly cache: string;
  private readonly indexer: Indexer;
  private readonly memory = new Map<string, Map<string, Definition> | Error>();
  // Every (sha, package key) a build asked for, so `phoropter index` knows what to write.
  readonly asked = new Map<string, Set<string>>();

  constructor(repo: Repo, opts: { cache?: string; indexer?: Indexer } = {}) {
    this.repo = repo;
    // One cache per repository, keyed by its path.
    const id = createHash('sha256').update(repo.dir).digest('hex').slice(0, 16);
    this.cache = join(opts.cache ?? cacheDir(), 'index', id);
    this.indexer = opts.indexer ?? defaultIndexer;
  }

  committedPath(sha: string, key: string): string {
    return join(this.repo.dir, STORE, sha, fileOf(split(key)[1]));
  }

  private cachedPath(sha: string, key: string): string {
    return join(this.cache, sha, fileOf(split(key)[1]));
  }

  isCommitted(sha: string, key: string): boolean {
    return this.readAt(this.committedPath(sha, key), key) !== undefined;
  }

  // The definitions at one commit for these package keys: from the committed
  // store, then the cache, then each scheme's indexer, run once for whatever
  // of its packages is missing, filling the cache.
  lookup(sha: string, keys: Set<string>): Lookup {
    const definitions = new Map<string, Definition>();
    const failed = new Map<string, string>();
    const missing = new Map<string, string[]>(); // by scheme
    for (const key of keys) {
      if (!this.asked.has(sha)) this.asked.set(sha, new Set());
      this.asked.get(sha)!.add(key);
      const known = this.memory.get(`${sha} ${key}`) ?? this.read(sha, key);
      if (known instanceof Error) failed.set(key, known.message);
      else if (known) for (const [k, d] of known) definitions.set(k, d);
      else {
        const [scheme, pkg] = split(key);
        if (!missing.has(scheme)) missing.set(scheme, []);
        missing.get(scheme)!.push(pkg);
      }
    }
    for (const [scheme, pkgs] of missing) {
      let found: Found | Error;
      try {
        found = this.indexer(this.repo, sha, scheme, pkgs);
      } catch (err) {
        found = new Error((err as Error).message.split('\n')[0]);
      }
      for (const pkg of pkgs) {
        const key = `${scheme} ${pkg}`;
        const mine = found instanceof Error ? found : (found.packages.get(pkg) ?? new Error(`${found.indexer} found no package ${pkg}`));
        this.memory.set(`${sha} ${key}`, mine);
        if (mine instanceof Error) {
          failed.set(key, mine.message);
          continue;
        }
        this.write(this.cachedPath(sha, key), serialize(found as Found, pkg, mine));
        for (const [k, d] of mine) definitions.set(k, d);
      }
    }
    return { definitions, failed };
  }

  private readAt(path: string, key: string): Map<string, Definition> | undefined {
    if (!existsSync(path)) return undefined;
    const { scheme, defs } = deserialize(readFileSync(path, 'utf8'));
    return scheme === split(key)[0] ? defs : undefined;
  }

  private read(sha: string, key: string): Map<string, Definition> | undefined {
    for (const path of [this.committedPath(sha, key), this.cachedPath(sha, key)]) {
      const defs = this.readAt(path, key);
      if (!defs) continue;
      this.memory.set(`${sha} ${key}`, defs);
      return defs;
    }
    return undefined;
  }

  // `phoropter index`: commits the index of one (sha, package key) into the
  // repository, from the cache or by indexing. Returns the path written, or
  // undefined when it was already there or couldn't be indexed.
  commit(sha: string, key: string): string | undefined {
    const path = this.committedPath(sha, key);
    if (this.isCommitted(sha, key)) return undefined;
    this.lookup(sha, new Set([key]));
    const cached = this.cachedPath(sha, key);
    if (!existsSync(cached)) return undefined;
    this.write(path, readFileSync(cached, 'utf8'));
    return path;
  }

  private write(path: string, text: string): void {
    mkdirSync(join(path, '..'), { recursive: true });
    writeFileSync(path, text);
  }
}
