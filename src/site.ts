// Builds the site model from one repo at one commit: its plans, their
// as-built files, each anchor shown at the entry's own commit and at the
// generated one, and the reverse pages ("places") the anchors point at.

import { basename } from 'node:path';
import type { Repo } from './git.ts';
import { parsePlan, slugify, type Anchor, type Entry, type PlanDoc } from './plan.ts';
import { indexCommit, packageName, type Definition } from './scip.ts';
import { resolveAt, target, type Hunk, type Target, type View } from './resolve.ts';

export type Commit = { sha: string; short: string; role: string };

export type ResolvedAnchor = {
  anchor: Anchor;
  target: Target;
  place?: Place;
  views: View[];
  note?: string; // for anchors that are shown but not resolved
};

export type ResolvedEntry = Entry & { resolved: ResolvedAnchor[] };

export type Doc = {
  file: string; // repo-relative markdown path
  href: string; // site-relative directory, ending in /
  doc: PlanDoc;
  before?: Commit; // base for a plan, head for an as-built
  beforeText: string; // as written, in case it isn't a commit
  entries: ResolvedEntry[];
};

export type Plan = Doc & { cart: string; asBuilt?: Doc };

export type Place = {
  key: string;
  kind: 'symbol' | 'file';
  label: string; // what the page is about, short
  full: string; // the symbol or path in full
  href: string;
  definition?: Definition; // at the generated commit, for symbols
  code?: Hunk; // the whole definition at the generated commit
  refs: { plan: Plan; doc: Doc; entry: ResolvedEntry; anchor: Anchor }[];
};

export type Site = {
  repoName: string;
  generated: Commit;
  plans: Plan[];
  places: Place[];
};

export function buildSite(repo: Repo, ref: string): Site {
  const sha = repo.resolve(ref);
  if (!sha) throw new Error(`not a commit in ${repo.dir}: ${ref}`);
  const repoName = basename(repo.dir.replace(/\/+$/, ''));
  const generated: Commit = { sha, short: repo.short(sha), role: 'generated' };

  const files = repo.files(sha, 'plans').filter((f) => /^plans\/[^/]+\.md$/.test(f));
  const planFiles = files.filter((f) => !f.endsWith('.as-built.md') && !f.endsWith('.notes.md'));

  const plans: Plan[] = planFiles.sort().map((file) => {
    const doc = parsePlan(repo.show(sha, file)!);
    const cart = doc.frontmatter.cart || basename(file, '.md').replace(/^\d{4}-\d{2}-\d{2}-/, '');
    const href = `plans/${slugify(cart)}/`;
    const plan: Plan = { file, href, doc, cart, ...commitOf(repo, doc.frontmatter.base, 'base'), entries: [] };
    const asBuiltFile = file.replace(/\.md$/, '.as-built.md');
    if (files.includes(asBuiltFile)) {
      const abDoc = parsePlan(repo.show(sha, asBuiltFile)!);
      plan.asBuilt = { file: asBuiltFile, href: `${href}as-built/`, doc: abDoc, ...commitOf(repo, abDoc.frontmatter.head, 'as-built head'), entries: [] };
    }
    return plan;
  });

  // Index each commit that a SCIP anchor needs, once.
  const docs = plans.flatMap((p) => (p.asBuilt ? [p, p.asBuilt] : [p]));
  const packages = new Set<string>();
  for (const d of docs) {
    for (const e of d.doc.entries) {
      for (const a of e.anchors) {
        const t = target(a, repoName);
        if (t.kind === 'symbol' && !a.isNew) packages.add(packageName(t.symbol)!);
      }
    }
  }
  const indexes = new Map<string, Map<string, Definition>>();
  const indexFor = (c: Commit) => {
    if (packages.size === 0) return undefined;
    if (!indexes.has(c.sha)) indexes.set(c.sha, indexCommit(repo, c.sha, packages));
    return indexes.get(c.sha);
  };

  const places = new Map<string, Place>();
  for (const plan of plans) {
    for (const d of plan.asBuilt ? [plan, plan.asBuilt] : [plan]) {
      const commits = d.before && d.before.sha !== sha ? [d.before, generated] : [generated];
      d.entries = d.doc.entries.map((entry) => {
        const re: ResolvedEntry = { ...entry, resolved: [] };
        re.resolved = entry.anchors.map((anchor) => {
          const t = target(anchor, repoName);
          const ra: ResolvedAnchor = { anchor, target: t, views: [] };
          if (t.kind === 'cross-repo') {
            ra.note = `in ${t.repo}, which this site doesn't read`;
            return ra;
          }
          ra.place = placeFor(places, t, indexFor(generated));
          ra.place.refs.push({ plan, doc: d, entry: re, anchor });
          if (anchor.isNew) {
            ra.note = 'made by the work, so there is nothing to resolve';
            return ra;
          }
          if (t.kind === 'file' && anchor.quote === undefined) {
            ra.note = 'no quote, so nothing to search for';
            return ra;
          }
          ra.views = commits.map((c) => resolveAt(repo, t, anchor.quote, c, t.kind === 'symbol' ? indexFor(c) : undefined));
          return ra;
        });
        return re;
      });
    }
  }

  for (const place of places.values()) {
    const def = place.definition;
    if (!def) continue;
    const lines = (repo.show(sha, def.file) ?? '').split('\n').slice(def.startLine - 1, def.endLine);
    place.code = { start: def.startLine, lines, marked: [] };
  }

  return {
    repoName,
    generated,
    plans,
    places: [...places.values()].sort((a, b) => a.href.localeCompare(b.href)),
  };
}

function commitOf(repo: Repo, text: string | undefined, role: string): { before?: Commit; beforeText: string } {
  const sha = text ? repo.resolve(text) : undefined;
  return { before: sha ? { sha, short: repo.short(sha), role } : undefined, beforeText: text ?? '' };
}

function placeFor(places: Map<string, Place>, t: Target, generatedIndex: Map<string, Definition> | undefined): Place {
  const key = t.kind === 'symbol' ? `symbol ${t.symbol}` : t.kind === 'file' ? `file ${t.file}` : '';
  let place = places.get(key);
  if (place) return place;
  if (t.kind === 'symbol') {
    const definition = generatedIndex?.get(t.symbol);
    const descriptor = t.symbol.split(' ').slice(4).join(' ');
    const name = descriptor.slice(descriptor.lastIndexOf('`/') + 2) || descriptor;
    const segments = name.split(/[#./`()[\]]+/).filter(Boolean);
    const href = definition
      ? `symbols/${definition.file}/${segments.join('/')}/`
      : `symbols/not-at-generated/${slugify(t.symbol)}/`;
    place = { key, kind: 'symbol', label: name, full: t.symbol, href, definition, refs: [] };
  } else if (t.kind === 'file') {
    place = { key, kind: 'file', label: t.file, full: t.file, href: `files/${t.file}/`, refs: [] };
  } else {
    throw new Error('cross-repo anchors have no place');
  }
  places.set(key, place);
  return place;
}
