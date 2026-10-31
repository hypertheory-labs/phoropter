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
// commits and git stores them once.

import { createHash } from 'node:crypto';
import { existsSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { homedir } from 'node:os';
import { join } from 'node:path';
import type { Repo } from './git.ts';
import { INDEXER, indexCommit, packageName, type Definition } from './scip.ts';

export const STORE = '.phoropter/index';

type Stored = {
  indexer: string;
  version: string;
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

export function serialize(pkg: string, defs: Map<string, Definition>): string {
  const definitions: Stored['definitions'] = {};
  for (const key of [...defs.keys()].sort()) {
    const d = defs.get(key)!;
    definitions[key] = [d.file, d.startLine, d.endLine];
  }
  const stored: Stored = { indexer: INDEXER.name, version: INDEXER.version, package: pkg, definitions };
  return JSON.stringify(stored, null, 1) + '\n';
}

function deserialize(text: string): Map<string, Definition> {
  const stored = JSON.parse(text) as Stored;
  const defs = new Map<string, Definition>();
  for (const [key, [file, startLine, endLine]] of Object.entries(stored.definitions)) {
    defs.set(key, { symbol: key, file, startLine, endLine });
  }
  return defs;
}

export type Indexer = (repo: Repo, sha: string, packages: Set<string>) => Map<string, Definition>;

// What resolving at one commit gets: the definitions found, and the packages
// that couldn't be indexed (their anchors show `no-index`).
export type Lookup = { definitions: Map<string, Definition>; failed: Map<string, string> };

export class Store {
  readonly repo: Repo;
  readonly cache: string;
  private readonly indexer: Indexer;
  private readonly memory = new Map<string, Map<string, Definition> | Error>();
  // Every (sha, package) a build asked for, so `phoropter index` knows what to write.
  readonly asked = new Map<string, Set<string>>();

  constructor(repo: Repo, opts: { cache?: string; indexer?: Indexer } = {}) {
    this.repo = repo;
    // One cache per repository, keyed by its path.
    const id = createHash('sha256').update(repo.dir).digest('hex').slice(0, 16);
    this.cache = join(opts.cache ?? cacheDir(), 'index', id);
    this.indexer = opts.indexer ?? indexCommit;
  }

  committedPath(sha: string, pkg: string): string {
    return join(this.repo.dir, STORE, sha, fileOf(pkg));
  }

  private cachedPath(sha: string, pkg: string): string {
    return join(this.cache, sha, fileOf(pkg));
  }

  isCommitted(sha: string, pkg: string): boolean {
    return existsSync(this.committedPath(sha, pkg));
  }

  // The definitions at one commit for these packages: from the committed
  // store, then the cache, then the indexer, which runs once for whatever is
  // missing and fills the cache.
  lookup(sha: string, packages: Set<string>): Lookup {
    const definitions = new Map<string, Definition>();
    const failed = new Map<string, string>();
    const missing = new Set<string>();
    for (const pkg of packages) {
      if (!this.asked.has(sha)) this.asked.set(sha, new Set());
      this.asked.get(sha)!.add(pkg);
      const known = this.memory.get(`${sha} ${pkg}`) ?? this.read(sha, pkg);
      if (known instanceof Error) failed.set(pkg, known.message);
      else if (known) for (const [k, d] of known) definitions.set(k, d);
      else missing.add(pkg);
    }
    if (missing.size) {
      let found: Map<string, Definition> | Error;
      try {
        found = this.indexer(this.repo, sha, missing);
      } catch (err) {
        found = new Error(`${INDEXER.name} couldn't index it: ${(err as Error).message.split('\n')[0]}`);
      }
      for (const pkg of missing) {
        if (found instanceof Error) {
          this.memory.set(`${sha} ${pkg}`, found);
          failed.set(pkg, found.message);
          continue;
        }
        const mine = new Map([...found].filter(([k]) => packageName(k) === pkg));
        this.memory.set(`${sha} ${pkg}`, mine);
        this.write(this.cachedPath(sha, pkg), serialize(pkg, mine));
        for (const [k, d] of mine) definitions.set(k, d);
      }
    }
    return { definitions, failed };
  }

  private read(sha: string, pkg: string): Map<string, Definition> | undefined {
    for (const path of [this.committedPath(sha, pkg), this.cachedPath(sha, pkg)]) {
      if (!existsSync(path)) continue;
      const defs = deserialize(readFileSync(path, 'utf8'));
      this.memory.set(`${sha} ${pkg}`, defs);
      return defs;
    }
    return undefined;
  }

  // `phoropter index`: commits the index of one (sha, package) into the
  // repository, from the cache or by indexing. Returns the path written, or
  // undefined when it was already there or couldn't be indexed.
  commit(sha: string, pkg: string): string | undefined {
    const path = this.committedPath(sha, pkg);
    if (existsSync(path)) return undefined;
    const { definitions, failed } = this.lookup(sha, new Set([pkg]));
    if (failed.has(pkg)) return undefined;
    this.write(path, serialize(pkg, definitions));
    return path;
  }

  private write(path: string, text: string): void {
    mkdirSync(join(path, '..'), { recursive: true });
    writeFileSync(path, text);
  }
}
