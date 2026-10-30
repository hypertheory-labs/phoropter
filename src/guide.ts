// The guide describes the tool, not the repository being generated, so it's
// read from phoropter itself and says which phoropter it came from.

import { execFileSync } from 'node:child_process';
import { readFileSync } from 'node:fs';

export type Guide = { markdown: string; version: string };

const root = new URL('../', import.meta.url);

export function readGuide(): Guide {
  return { markdown: readFileSync(new URL('guide.md', root), 'utf8'), version: version() };
}

// The tool's commit when it runs from a checkout, saying so when the guide
// has changes that commit doesn't have; otherwise its package version.
function version(): string {
  const dir = root.pathname;
  try {
    const sha = execFileSync('git', ['-C', dir, 'rev-parse', '--short', 'HEAD'], { encoding: 'utf8', stdio: ['ignore', 'pipe', 'ignore'] }).trim();
    const dirty = execFileSync('git', ['-C', dir, 'status', '--porcelain', '--', 'guide.md'], { encoding: 'utf8' }).trim();
    return dirty ? `commit ${sha}, with uncommitted changes to the guide` : `commit ${sha}`;
  } catch {
    const pkg = JSON.parse(readFileSync(new URL('package.json', root), 'utf8'));
    return `version ${pkg.version}`;
  }
}
