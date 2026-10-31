// Finding and running indexers (seersucker). An indexer is any program that,
// given a checked-out repository and package names, prints definitions as
// JSON; indexers/README.md is the contract. The scheme an anchor's symbol
// starts with picks the indexer:
//
// 1. a first-party one in indexers/<name>/, whose indexer.json gives the
//    command that runs it (and whose --describe names its scheme);
// 2. otherwise a program on PATH named phoropter-index-<scheme>, the way gh
//    and kubectl find extensions. Nothing to configure.

import { spawnSync } from 'node:child_process';
import { existsSync, readdirSync, readFileSync, statSync } from 'node:fs';
import { delimiter, join } from 'node:path';
import type { Definition } from './scip.ts';

export type IndexerInfo = { indexer: string; version: string; scheme: string; extensions: string[]; command: string[]; cwd?: string };

export type IndexerOutput = {
  indexer: string;
  version: string;
  scheme: string;
  packages: Record<string, { definitions: Record<string, [string, number, number]> }>;
  missing?: string[];
};

const FIRST_PARTY = new URL('../indexers/', import.meta.url).pathname;

let known: IndexerInfo[] | undefined;

// What a program says about itself with --describe.
export function describe(command: string[], cwd?: string): Omit<IndexerInfo, 'command' | 'cwd'> {
  const r = spawnSync(command[0], [...command.slice(1), '--describe'], { cwd, encoding: 'utf8' });
  if (r.error) throw new Error(`couldn't run ${command.join(' ')}: ${r.error.message}`);
  if (r.status !== 0) throw new Error(`${command.join(' ')} --describe failed: ${(r.stderr || '').trim().split('\n')[0]}`);
  const d = JSON.parse(r.stdout);
  return { indexer: d.indexer, version: d.version, scheme: d.scheme, extensions: d.extensions ?? [] };
}

// Every indexer this machine has: first-party ones, then any on PATH.
export function indexers(): IndexerInfo[] {
  if (known) return known;
  known = [];
  for (const name of existsSync(FIRST_PARTY) ? readdirSync(FIRST_PARTY) : []) {
    const dir = join(FIRST_PARTY, name);
    const manifest = join(dir, 'indexer.json');
    if (!existsSync(manifest)) continue;
    const { command } = JSON.parse(readFileSync(manifest, 'utf8')) as { command: string[] };
    try {
      known.push({ ...describe(command, dir), command, cwd: dir });
    } catch {
      // its toolchain isn't here; anchors in its scheme show no-index
    }
  }
  for (const dir of (process.env.PATH ?? '').split(delimiter)) {
    let names: string[] = [];
    try {
      names = readdirSync(dir).filter((n) => n.startsWith('phoropter-index-'));
    } catch {
      continue;
    }
    for (const name of names) {
      const path = join(dir, name);
      if (known.some((k) => k.scheme === name.slice('phoropter-index-'.length)) || !statSync(path).isFile()) continue;
      try {
        known.push({ ...describe([path]), command: [path] });
      } catch {
        // not runnable here
      }
    }
  }
  return known;
}

export function indexerFor(scheme: string): IndexerInfo | undefined {
  return indexers().find((i) => i.scheme === scheme);
}

// Runs an indexer on a checked-out directory, for some packages (all of them
// when none are named). Throws with the indexer's own one line on failure.
export function runIndexer(info: Pick<IndexerInfo, 'command' | 'cwd'>, dir: string, packages: string[] = []): IndexerOutput {
  const r = spawnSync(info.command[0], [...info.command.slice(1), dir, ...packages], { cwd: info.cwd, encoding: 'utf8', maxBuffer: 1 << 30 });
  if (r.error) throw new Error(`couldn't run ${info.command.join(' ')}: ${r.error.message}`);
  if (r.status !== 0) throw new Error((r.stderr || `exited ${r.status}`).trim().split('\n').at(-1)!);
  return JSON.parse(r.stdout) as IndexerOutput;
}

export function definitionsOf(out: IndexerOutput, pkg: string): Map<string, Definition> | undefined {
  const p = out.packages[pkg];
  if (!p) return undefined;
  return new Map(Object.entries(p.definitions).map(([symbol, [file, startLine, endLine]]) => [symbol, { symbol, file, startLine, endLine }]));
}
