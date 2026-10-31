// The one draft plan, `drafts/plan.draft.md`, read from the working tree on
// every request (loden; tarlatan moved it into drafts/). Everything else on the
// site is read through git at a commit; a draft is uncommitted by nature, and
// says so.
//
// Its notes are `drafts/plan.draft.notes.md`, in the annotating agreement's
// format. A landed plan's notes are kept in drafts/ too, as
// `drafts/<plan's file name>.notes.md` (tarlatan): a person saying the plan
// matters to something now (organza).
// A draft's open questions are the agreement's prompts (Question, Select,
// Affirm) at any heading level, plus phoropter's `**If unanswered:**`.

import { createHash } from 'node:crypto';
import { existsSync, readFileSync } from 'node:fs';
import { basename, join } from 'node:path';
import { parsePlan } from './plan.ts';
import { commitOf, packagesOf, resolveEntries, type Commit, type Plan, type Site } from './site.ts';

export const DRAFTS = 'drafts';
export const DRAFT = 'drafts/plan.draft.md';
export const NOTES = 'drafts/plan.draft.notes.md';

// A landed plan's notes file, whether or not it exists.
export function planNotesFile(plan: Plan): string {
  return `${DRAFTS}/${basename(plan.file, '.md')}.notes.md`;
}

// A landed plan's notes, each placed after the entry whose source holds its
// quote. Undefined when there are none.
export function readPlanNotes(dir: string, plan: Plan): Annotation[] | undefined {
  const path = join(dir, planNotesFile(plan));
  if (!existsSync(path)) return undefined;
  const planPath = join(dir, plan.file);
  const source = existsSync(planPath) ? readFileSync(planPath, 'utf8') : '';
  return parseNotes(readFileSync(path, 'utf8')).map((a) => placed(a, source, plan));
}

export type Prompt = {
  kind: string; // Question, Select, or Affirm
  text: string; // the heading after the keyword
  body: string; // markdown, without the options or the fallback
  options: string[]; // a Select's first list
  fallback?: string; // markdown, after **If unanswered:**
  answer?: Annotation; // from the notes, which restate the prompt's heading
};

export type Annotation = {
  heading: string; // as written: a location and author, or a restated prompt
  quote?: string; // the source text it quotes, as written
  comment: string; // markdown
  at?: string; // the slug of the entry holding the quote, or 'intro'
  found: boolean; // false when the quote is in the draft nowhere
};

export type Draft = {
  plan: Plan;
  readAt: Date;
  sha256: string;
  prompts: Map<string, Prompt[]>; // by entry slug, or 'intro'
  notes?: { sha256: string; annotates?: string; current: boolean; annotations: Annotation[] };
};

export function hasDraft(dir: string): boolean {
  return existsSync(join(dir, DRAFT));
}

export function readDraft(site: Site, dir: string): Draft | undefined {
  const path = join(dir, DRAFT);
  if (!existsSync(path)) return undefined;
  const source = readFileSync(path, 'utf8');
  const cx = site.context;
  const doc = parsePlan(source);

  const prompts = new Map<string, Prompt[]>();
  const intro = splitPrompts(doc.intro);
  doc.intro = intro.rest;
  if (intro.prompts.length) prompts.set('intro', intro.prompts);
  for (const e of doc.entries) {
    const split = splitPrompts(e.body);
    e.body = split.rest;
    if (split.prompts.length) prompts.set(e.slug, split.prompts);
  }

  const plan: Plan = { file: DRAFT, href: 'draft/', doc, name: 'draft', ...commitOf(cx.repo, doc.frontmatter.base, 'base'), entries: [], revisitedBy: [] };
  plan.revisits = site.plans.find((p) => p.file === doc.frontmatter.revisits);
  plan.entries = doc.entries.map((entry) => ({ ...entry, resolved: [], answeredBy: [], anchoredBy: [] }));
  const commits: Commit[] = [cx.generated];
  if (plan.before && plan.before.sha !== cx.generated.sha) commits.unshift(plan.before);
  const packages = new Set([...packagesOf(cx, cx.docs), ...packagesOf(cx, [plan])]);
  resolveEntries(cx, plan, plan, commits, packages, false);

  const draft: Draft = { plan, readAt: new Date(), sha256: sha256(source), prompts };
  const notesPath = join(dir, NOTES);
  if (existsSync(notesPath)) {
    const text = readFileSync(notesPath, 'utf8');
    const annotations = parseNotes(text);
    const all = [...prompts.values()].flat();
    const rest: Annotation[] = [];
    for (const a of annotations) {
      const restated = /^(Question|Select|Affirm):\s*(.*)$/i.exec(a.heading);
      const prompt = restated && all.find((p) => p.kind.toLowerCase() === restated[1].toLowerCase() && p.text === restated[2].trim());
      if (prompt) prompt.answer = a;
      else rest.push(placed(a, source, plan));
    }
    const annotates = /annotates: \S+ sha256:([0-9a-f]+)/.exec(text)?.[1];
    // annotations.nvim hashes the buffer's lines joined by newlines, which
    // drops the file's final newline; either form counts as this version.
    const current = !!annotates && [source, source.replace(/\n$/, '')].some((v) => sha256(v).startsWith(annotates));
    draft.notes = { sha256: sha256(text), annotates, current, annotations: rest };
  }
  return draft;
}

// Takes each prompt out of a block of markdown. A prompt runs until the next
// heading of the same or a higher level, as the agreement says.
export function splitPrompts(md: string): { rest: string; prompts: Prompt[] } {
  const lines = md.split('\n');
  const prompts: Prompt[] = [];
  const rest: string[] = [];
  for (let i = 0; i < lines.length; i++) {
    const m = /^(#{1,6})\s+(Question|Select|Affirm):\s*(.*)$/i.exec(lines[i]);
    if (!m) {
      rest.push(lines[i]);
      continue;
    }
    const level = m[1].length;
    let end = i + 1;
    while (end < lines.length && !(/^(#{1,6})\s/.exec(lines[end]) && /^(#{1,6})\s/.exec(lines[end])![1].length <= level)) end++;
    prompts.push(promptOf(m[2], m[3].trim(), lines.slice(i + 1, end)));
    i = end - 1;
  }
  return { rest: rest.join('\n').trim(), prompts };
}

function promptOf(keyword: string, text: string, lines: string[]): Prompt {
  const kind = keyword[0].toUpperCase() + keyword.slice(1).toLowerCase();
  const prompt: Prompt = { kind, text, body: '', options: [] };
  const body: string[] = [];
  let inOptions = false;
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const fallback = /^\*\*If unanswered:\*\*\s*(.*)$/.exec(line);
    if (fallback) {
      const para = [fallback[1]];
      while (i + 1 < lines.length && lines[i + 1].trim()) para.push(lines[++i]);
      prompt.fallback = para.join('\n').trim();
      continue;
    }
    const option = /^[-*]\s+(.*)$/.exec(line);
    if (kind === 'Select' && option && (inOptions || !prompt.options.length)) {
      inOptions = true;
      prompt.options.push(option[1]);
      continue;
    }
    if (inOptions && line.trim()) inOptions = false;
    body.push(line);
  }
  prompt.body = body.join('\n').trim();
  return prompt;
}

// The notes file: `## heading`, an optional blockquote of the source, then the
// comment. The title and HTML comments before the first entry are skipped.
export function parseNotes(text: string): Annotation[] {
  const annotations: Annotation[] = [];
  let current: { heading: string; lines: string[] } | undefined;
  const flush = () => {
    if (!current) return;
    const lines = current.lines;
    while (lines.length && !lines[0].trim()) lines.shift();
    const quote: string[] = [];
    while (lines.length && /^>/.test(lines[0])) quote.push(lines.shift()!.replace(/^> ?/, ''));
    annotations.push({ heading: current.heading, quote: quote.length ? quote.join('\n') : undefined, comment: lines.join('\n').trim(), found: true });
  };
  for (const line of text.split('\n')) {
    const h = /^## (.*)$/.exec(line);
    if (h) {
      flush();
      current = { heading: h[1].trim(), lines: [] };
    } else if (current) {
      current.lines.push(line);
    }
  }
  flush();
  return annotations;
}

// Finds which entry's source holds the quote. A quote elided with `…` on its
// own line matches its first part, then its last part after it.
function placed(a: Annotation, source: string, plan: Plan): Annotation {
  if (a.quote === undefined) return a;
  const [first, ...more] = a.quote.split(/\n…\n/);
  const at = source.indexOf(first);
  const last = more.at(-1);
  if (at === -1 || (last !== undefined && source.indexOf(last, at + first.length) === -1)) return { ...a, found: false };
  // Entry headings are `## ` lines outside code fences, in the same order
  // as the parsed entries.
  let offset = 0;
  let fenced = false;
  let entry = -1;
  for (const line of source.split('\n')) {
    if (offset > at) break;
    if (/^(```|~~~)/.test(line)) fenced = !fenced;
    else if (!fenced && /^## /.test(line)) entry++;
    offset += line.length + 1;
  }
  return { ...a, at: entry === -1 ? 'intro' : plan.entries[entry]?.slug ?? 'intro' };
}

function sha256(text: string): string {
  return createHash('sha256').update(text).digest('hex');
}
