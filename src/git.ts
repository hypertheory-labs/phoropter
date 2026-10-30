// Everything read from the repo goes through git at a commit, never the
// working tree, so the site is a function of the commit alone.

import { execFileSync } from 'node:child_process';
import { mkdtempSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

export class Repo {
  readonly dir: string;
  constructor(dir: string) {
    this.dir = dir;
  }

  git(...args: string[]): string {
    return execFileSync('git', ['-C', this.dir, ...args], { encoding: 'utf8', maxBuffer: 64 * 1024 * 1024 });
  }

  // The full sha, or undefined when `ref` isn't a commit here (a plan's base
  // can be prose, like gimlet's "none, a new repo").
  resolve(ref: string): string | undefined {
    try {
      return this.git('rev-parse', '--verify', '--quiet', `${ref}^{commit}`).trim();
    } catch {
      return undefined;
    }
  }

  // The shortest unambiguous form, at least 7 characters, the way git shows it.
  short(sha: string): string {
    return this.git('rev-parse', '--short', sha).trim();
  }

  files(sha: string, dir = ''): string[] {
    const args = ['ls-tree', '-r', '--name-only', sha];
    if (dir) args.push('--', dir);
    return this.git(...args).split('\n').filter(Boolean);
  }

  // The commit that first added `path`, as seen from `sha`, and its time.
  added(sha: string, path: string): { sha: string; time: number } | undefined {
    const first = this.git('log', '--diff-filter=A', '--format=%H %ct', '--reverse', sha, '--', path).split('\n')[0];
    if (!first) return undefined;
    const [full, time] = first.split(' ');
    return { sha: full, time: Number(time) };
  }

  show(sha: string, path: string): string | undefined {
    try {
      return execFileSync('git', ['-C', this.dir, 'show', `${sha}:${path}`], {
        encoding: 'utf8',
        maxBuffer: 64 * 1024 * 1024,
        stdio: ['ignore', 'pipe', 'ignore'],
      });
    } catch {
      return undefined;
    }
  }

  // Writes the tree at `sha` into a fresh temp directory, for the indexer.
  // The caller removes it.
  checkout(sha: string): string {
    const dir = mkdtempSync(join(tmpdir(), 'phoropter-'));
    execFileSync('sh', ['-c', `git -C "$1" archive "$2" | tar -x -C "$3"`, 'sh', this.dir, sha, dir]);
    return dir;
  }
}

export function removeDir(dir: string): void {
  rmSync(dir, { recursive: true, force: true });
}
