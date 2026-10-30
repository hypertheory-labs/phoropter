// Builds the site model from one repo at one commit: its plans, their
// as-built files, each anchor shown at the entry's own commit and at the
// generated one, and the reverse pages ("places") the anchors point at.

import { basename } from 'node:path';
import type { Repo } from './git.ts';
import { parsePlan, slugify, type Anchor, type Entry, type PlanDoc } from './plan.ts';
import { indexCommit, packageName, type Definition } from './scip.ts';
import { collapse, resolveAt, resolveInEntry, target, type Hunk, type Target, type View } from './resolve.ts';

export type Commit = { sha: string; short: string; role: string };

export type ResolvedAnchor = {
  anchor: Anchor;
  target: Target;
  place?: Place;
  entry?: EntryRef; // an anchor at another plan's entry
  plan?: Plan; // an anchor at a whole plan or as-built file
  views: View[];
  note?: string; // for anchors that are shown but not resolved
};

export type EntryRef = { plan: Plan; doc: Doc; entry: ResolvedEntry };

export type ResolvedEntry = Entry & {
  resolved: ResolvedAnchor[];
  answers?: EntryRef; // an Answered, to its plan's Guess of the same title
  answeredBy: EntryRef[]; // a Guess, to the answers in its as-built
  anchoredBy: EntryRef[]; // entries in other plans whose anchors name this one
};

export type Doc = {
  file: string; // repo-relative markdown path
  href: string; // site-relative directory, ending in /
  doc: PlanDoc;
  before?: Commit; // base for a plan, head for an as-built
  beforeText: string; // as written, in case it isn't a commit
  entries: ResolvedEntry[];
};

export type Plan = Doc & {
  name: string; // from the filename: `2026-10-29-pumice.md` is `pumice`
  date?: string; // from the filename too, which is the only place it's written
  added?: Commit & { time: number }; // the commit that added the file; what orders plans
  asBuilt?: Doc;
  revisits?: Plan; // bodkin's `revisits:`, when it names a plan on this site
  revisitedBy: Plan[];
};

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

  const plans: Plan[] = planFiles.map((file) => {
    const doc = parsePlan(repo.show(sha, file)!);
    // The filename, not the cart: a cart is optional (lanyard), and plans
    // without one would share a page.
    const name = basename(file, '.md').replace(/^\d{4}-\d{2}-\d{2}-/, '');
    const href = `plans/${slugify(name)}/`;
    const plan: Plan = { file, href, doc, name, ...commitOf(repo, doc.frontmatter.base, 'base'), entries: [], revisitedBy: [] };
    plan.date = /^(\d{4}-\d{2}-\d{2})-/.exec(basename(file))?.[1];
    const added = repo.added(sha, file);
    if (added) plan.added = { sha: added.sha, short: repo.short(added.sha), role: 'added', time: added.time };
    const asBuiltFile = file.replace(/\.md$/, '.as-built.md');
    if (files.includes(asBuiltFile)) {
      const abDoc = parsePlan(repo.show(sha, asBuiltFile)!);
      plan.asBuilt = { file: asBuiltFile, href: `${href}as-built/`, doc: abDoc, ...commitOf(repo, abDoc.frontmatter.head, 'as-built head'), entries: [] };
    }
    return plan;
  });

  // Oldest first: by the commit that added each plan, since several plans can
  // share a date. The filename breaks a tie.
  plans.sort((a, b) => (a.added?.time ?? 0) - (b.added?.time ?? 0) || a.file.localeCompare(b.file));

  for (const plan of plans) {
    const revisits = plans.find((p) => p.file === plan.doc.frontmatter.revisits);
    if (revisits) {
      plan.revisits = revisits;
      revisits.revisitedBy.push(plan);
    }
  }

  // Every entry exists before any anchor resolves, since an anchor can name
  // an entry in a plan that comes later.
  const docs = plans.flatMap((p) => (p.asBuilt ? [p, p.asBuilt] : [p]));
  const planOf = new Map<Doc, Plan>(plans.flatMap((p) => (p.asBuilt ? [[p, p], [p.asBuilt, p]] : [[p, p]])));
  const onSite = new Set(docs.map((d) => d.file));
  for (const d of docs) d.entries = d.doc.entries.map((entry) => ({ ...entry, resolved: [], answeredBy: [], anchoredBy: [] }));

  // Index each commit that a SCIP anchor needs, once.
  const packages = new Set<string>();
  for (const d of docs) {
    for (const e of d.doc.entries) {
      for (const a of e.anchors) {
        const t = target(a, repoName, onSite);
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
      // A plan's anchors show at its base, at its as-built's head (where its own
      // work landed, so lanyard's drift starts there), and at the generated
      // commit. An as-built's show at its head and the generated commit.
      const head = d === plan ? plan.asBuilt?.before : undefined;
      const commits: Commit[] = [generated];
      if (head && head.sha !== sha) commits.unshift(head);
      if (d.before && !commits.some((c) => c.sha === d.before!.sha)) commits.unshift(d.before);
      for (const re of d.entries) {
        re.resolved = re.anchors.map((anchor) => {
          const t = target(anchor, repoName, onSite);
          const ra: ResolvedAnchor = { anchor, target: t, views: [] };
          if (t.kind === 'cross-repo') {
            ra.note = `in ${t.repo}, which this site doesn't read`;
            return ra;
          }
          if (t.kind === 'plan') {
            const doc = docs.find((x) => x.file === t.file)!;
            const owner = planOf.get(doc)!;
            const where = doc === owner ? owner.name : `${owner.name}'s as-built`;
            if (t.title === undefined) {
              ra.plan = owner;
              if (anchor.quote !== undefined && !anchor.isNew) {
                ra.views = collapse(commits.map((c) => resolveAt(repo, { kind: 'file', file: t.file, path: [] }, anchor.quote, c, undefined)));
              }
              return ra;
            }
            const entry = doc.entries.find((e) => e.title === t.title);
            if (!entry) {
              ra.note = `no entry titled this in ${where}`;
              return ra;
            }
            ra.entry = { plan: owner, doc, entry };
            entry.anchoredBy.push({ plan, doc: d, entry: re });
            if (anchor.quote !== undefined) ra.views = [resolveInEntry(repo, t.file, t.title, anchor.quote, generated)];
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
          ra.views = collapse(commits.map((c) => resolveAt(repo, t, anchor.quote, c, t.kind === 'symbol' ? indexFor(c) : undefined)));
          return ra;
        });
      }
    }
  }

  // Grommet titles an answer with its guess's own title, so they pair by
  // exact title.
  for (const plan of plans) {
    for (const answer of plan.asBuilt?.entries ?? []) {
      if (answer.kind !== 'Answered') continue;
      const guess = plan.entries.find((e) => e.kind === 'Guess' && e.title === answer.title);
      if (!guess) continue;
      answer.answers = { plan, doc: plan, entry: guess };
      guess.answeredBy.push({ plan, doc: plan.asBuilt!, entry: answer });
    }
  }

  for (const place of places.values()) {
    const def = place.definition;
    if (!def) continue;
    const lines = (repo.show(sha, def.file) ?? '').split('\n').slice(def.startLine - 1, def.endLine);
    place.code = { start: def.startLine, lines, marked: [] };
  }

  // A reverse page lists newest first: someone arriving at a file usually
  // wants what touched it last.
  for (const place of places.values()) {
    place.refs.sort((a, b) => (b.plan.added?.time ?? 0) - (a.plan.added?.time ?? 0));
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
