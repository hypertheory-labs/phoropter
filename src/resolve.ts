// Shows an anchor's code at one commit. SCIP symbols go through the index;
// everything else is a quote searched for in the anchor's own file only.
// Nothing here judges a difference between two commits.

import type { Repo } from './git.ts';
import type { Anchor } from './plan.ts';
import { isScipSymbol, withoutVersion, type Definition } from './scip.ts';

export type Target =
  | { kind: 'symbol'; symbol: string } // version-less
  | { kind: 'plan'; file: string; title?: string } // a plan or as-built on this site, and an entry in it
  | { kind: 'file'; file: string; path: string[] } // path: heading or key path after the file
  | { kind: 'cross-repo'; repo: string; rest: string };

const CONTEXT = 3;
const DECLARATION = 2 * CONTEXT + 1; // as many lines as a found quote's view, at most

// `plans` is the plan and as-built files on this site; an anchor into one of
// them points at the plan, or at an entry by its title, not at a file.
export function target(anchor: Anchor, repoName: string, plans: Set<string> = new Set()): Target {
  const cross = /^([\w.-]+): (.+)$/.exec(anchor.where);
  if (cross && cross[1] !== repoName) return { kind: 'cross-repo', repo: cross[1], rest: cross[2] };
  const where = cross ? cross[2] : anchor.where;
  if (isScipSymbol(where)) return { kind: 'symbol', symbol: withoutVersion(where) };
  const [file, ...path] = where.split(' › ');
  if (plans.has(file.trim())) return { kind: 'plan', file: file.trim(), title: path.length ? path.join(' › ') : undefined };
  return { kind: 'file', file: file.trim(), path: path.map((p) => p.trim()) };
}

export type Hunk = {
  start: number; // 1-based line number of lines[0]
  lines: string[];
  marked: number[]; // line numbers the quote covers
};

export type View = {
  sha: string;
  short: string;
  role: string; // what this commit is to the entry: "base", "as-built head", "generated"
  status: 'found' | 'not-found' | 'no-quote' | 'no-file' | 'no-symbol';
  file?: string;
  startLine?: number; // the symbol's range, for SCIP anchors
  endLine?: number;
  matches: number; // how many times the quote matched in the file or range
  hunks: Hunk[];
  note: string; // plain words for the reader
  sameAs?: string; // short sha of the as-built head this view matches exactly; its code is not repeated
};

// When the generated view matches the as-built head's exactly, it says so and
// drops its code. A fact about two commits, not a judgment about drift.
export function collapse(views: View[]): View[] {
  const head = views.find((v) => v.role === 'as-built head');
  const last = views.at(-1);
  if (!head || !last || last === head || last.role !== 'generated') return views;
  const same = (v: View) => JSON.stringify([v.status, v.file, v.startLine, v.endLine, v.matches, v.hunks]);
  if (same(head) !== same(last)) return views;
  last.sameAs = head.short;
  last.note = `same as at \`${head.short}\``;
  last.hunks = [];
  return views;
}

export function resolveAt(
  repo: Repo,
  t: Target,
  quote: string | undefined,
  commit: { sha: string; short: string; role: string },
  definitions: Map<string, Definition> | undefined,
): View {
  const view: View = { ...commit, status: 'no-quote', matches: 0, hunks: [], note: '' };
  const needle = quote?.trim();

  if (t.kind === 'symbol') {
    const def = definitions?.get(t.symbol);
    if (!def) {
      view.status = 'no-symbol';
      view.note = `symbol not in the index at ${commit.short}`;
      return view;
    }
    const lines = (repo.show(commit.sha, def.file) ?? '').split('\n');
    view.file = def.file;
    view.startLine = def.startLine;
    view.endLine = def.endLine;
    const range = `lines ${def.startLine}–${def.endLine}`;
    const body = lines.slice(def.startLine - 1, def.endLine);
    const found = (needle ? findQuote(body, needle) : []).map((m) => m.map((n) => n + def.startLine - 1));
    view.matches = found.length;
    if (found.length) {
      view.status = 'found';
      view.hunks = found.map((m) => around(lines, m, def.startLine, def.endLine));
      view.note = `${range}; quote at ${describeLines(found.flat())}`;
      return view;
    }
    // No quote to centre on, so show the declaration: the symbol is still
    // here, even if the quote isn't. The whole range is on the symbol page.
    const end = Math.min(def.endLine, def.startLine + DECLARATION - 1);
    view.hunks = [{ start: def.startLine, lines: lines.slice(def.startLine - 1, end), marked: [] }];
    if (needle) {
      view.status = 'not-found';
      view.note = `${range}; quote not found at \`${commit.short}\`, so this is the declaration`;
    } else {
      view.note = `${range}; no quote, so this is the declaration`;
    }
    return view;
  }

  if (t.kind === 'cross-repo' || t.kind === 'plan') throw new Error(`${t.kind} anchors are not resolved here`);

  view.file = t.file;
  const text = repo.show(commit.sha, t.file);
  if (text === undefined) {
    view.status = 'no-file';
    view.note = `\`${t.file}\` doesn't exist at \`${commit.short}\``;
    return view;
  }
  if (!needle) {
    view.note = 'no quote, so nothing to search for';
    return view;
  }
  const lines = text.split('\n');
  const found = findQuote(lines, needle);
  view.matches = found.length;
  if (found.length === 0) {
    view.status = 'not-found';
    view.note = `quote not found at \`${commit.short}\``;
    return view;
  }
  view.status = 'found';
  view.hunks = found.map((m) => around(lines, m, 1, lines.length));
  const where = describeLines(found.flat());
  view.note = found.length === 1 ? `quote at ${where}` : `quote matches ${found.length} times, at ${where}`;
  return view;
}

// A quote inside one entry of a plan or as-built, at one commit. Both are
// frozen once their work lands (awl), so one commit is enough. The search
// stays inside the entry, from its heading to the next.
export function resolveInEntry(repo: Repo, file: string, title: string, quote: string, commit: { sha: string; short: string; role: string }): View {
  const view: View = { ...commit, status: 'not-found', file, matches: 0, hunks: [], note: '' };
  const lines = (repo.show(commit.sha, file) ?? '').split('\n');
  const heading = lines.findIndex((l) => /^## /.test(l) && l.replace(/^## (?:\w+: )?/, '') === title);
  if (heading === -1) {
    view.note = `no entry titled this at \`${commit.short}\``;
    return view;
  }
  let end = lines.findIndex((l, i) => i > heading && /^## /.test(l));
  if (end === -1) end = lines.length;
  while (end > heading + 1 && !lines[end - 1].trim()) end--;
  const found = findQuote(lines.slice(heading, end), quote.trim()).map((m) => m.map((n) => n + heading));
  view.matches = found.length;
  if (!found.length) {
    view.note = `quote not found in this entry at \`${commit.short}\``;
    return view;
  }
  view.status = 'found';
  view.hunks = [{ start: heading + 1, lines: lines.slice(heading, end), marked: found.flat() }];
  view.note = `quote at ${describeLines(found.flat())}`;
  return view;
}

// A match with CONTEXT lines either side, kept within lines lo..hi (1-based).
function around(lines: string[], match: number[], lo: number, hi: number): Hunk {
  const start = Math.max(lo, match[0] - CONTEXT);
  const end = Math.min(hi, match.at(-1)! + CONTEXT);
  return { start, lines: lines.slice(start - 1, end), marked: match };
}

// Each match is the list of 1-based line numbers (relative to `lines`) that it
// spans. A quote may span lines.
function findQuote(lines: string[], needle: string): number[][] {
  const text = lines.join('\n');
  const lineStarts = [0];
  for (let i = 0; i < text.length; i++) if (text[i] === '\n') lineStarts.push(i + 1);
  const lineOf = (offset: number) => {
    let lo = 0;
    let hi = lineStarts.length - 1;
    while (lo < hi) {
      const mid = (lo + hi + 1) >> 1;
      if (lineStarts[mid] <= offset) lo = mid;
      else hi = mid - 1;
    }
    return lo + 1;
  };
  const matches: number[][] = [];
  for (let at = text.indexOf(needle); at !== -1; at = text.indexOf(needle, at + 1)) {
    const first = lineOf(at);
    const last = lineOf(at + needle.length - 1);
    matches.push(Array.from({ length: last - first + 1 }, (_, i) => first + i));
  }
  return matches;
}

function describeLines(lines: number[]): string {
  const unique = [...new Set(lines)].sort((a, b) => a - b);
  return (unique.length === 1 ? 'line ' : 'lines ') + unique.join(', ');
}
