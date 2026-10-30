// Parses a plan or as-built file in the lanyard format: frontmatter, a title,
// intro prose, then `## Kind: title` entries, each optionally starting with a
// list of anchors (`where:`, `where (new):`, `quote:`) and followed by prose.

import { lexer, type Tokens } from 'marked';

export type Anchor = {
  where: string;
  isNew: boolean;
  quote?: string;
};

export type Entry = {
  kind: string; // Decision, Decline, Guess, Changed, Chose, Answered, or '' for a plain section
  title: string; // markdown, inline
  slug: string;
  anchors: Anchor[];
  body: string; // markdown
};

export type PlanDoc = {
  frontmatter: Record<string, string>;
  title: string;
  intro: string;
  entries: Entry[];
};

export function parsePlan(source: string): PlanDoc {
  const { frontmatter, rest } = splitFrontmatter(source);
  const tokens = lexer(rest);
  const doc: PlanDoc = { frontmatter, title: '', intro: '', entries: [] };
  let current: Entry | undefined;
  const slugs = new Set<string>();

  for (const token of tokens) {
    if (token.type === 'heading' && token.depth === 1 && !doc.title) {
      doc.title = token.text;
      continue;
    }
    if (token.type === 'heading' && token.depth === 2) {
      const match = /^(Decision|Decline|Guess|Changed|Chose|Answered):\s*(.*)$/.exec(token.text);
      const title = match ? match[2] : token.text;
      current = { kind: match ? match[1] : '', title, slug: uniqueSlug(title, slugs), anchors: [], body: '' };
      doc.entries.push(current);
      continue;
    }
    if (!current) {
      doc.intro += token.raw;
      continue;
    }
    if (token.type === 'list' && !current.body.trim() && current.anchors.length === 0) {
      const anchors = parseAnchors(token as Tokens.List);
      if (anchors) {
        current.anchors = anchors;
        continue;
      }
    }
    current.body += token.raw;
  }
  doc.intro = doc.intro.trim();
  for (const entry of doc.entries) entry.body = entry.body.trim();
  return doc;
}

function splitFrontmatter(source: string): { frontmatter: Record<string, string>; rest: string } {
  const match = /^---\n([\s\S]*?)\n---\n?/.exec(source);
  if (!match) return { frontmatter: {}, rest: source };
  const frontmatter: Record<string, string> = {};
  for (const line of match[1].split('\n')) {
    const kv = /^([\w-]+):\s*(.*)$/.exec(line);
    if (kv) frontmatter[kv[1]] = kv[2];
  }
  return { frontmatter, rest: source.slice(match[0].length) };
}

// An anchor list is one whose every item is `where:`, `where (new):`, or
// `quote:`. A `quote:` belongs to the `where` before it. Returns undefined for
// any other list, which is then just prose.
function parseAnchors(list: Tokens.List): Anchor[] | undefined {
  const anchors: Anchor[] = [];
  for (const item of list.items) {
    const match = /^(where \(new\)|where|quote):\s*([\s\S]*)$/.exec(item.text.trim());
    if (!match) return undefined;
    const value = unwrapCodeSpan(match[2].trim());
    if (match[1] === 'quote') {
      const last = anchors.at(-1);
      if (!last || last.quote !== undefined) return undefined;
      last.quote = value;
    } else {
      anchors.push({ where: value, isNew: match[1] === 'where (new)' });
    }
  }
  return anchors;
}

// CommonMark code span: a run of backticks, the content, the same run. One
// leading and trailing space are stripped when both are present, which is how
// a span can start or end with a backtick (`` `x` ``).
export function unwrapCodeSpan(text: string): string {
  const match = /^(`+)([\s\S]*)\1$/.exec(text);
  if (!match) return text;
  const inner = match[2];
  if (inner.length >= 2 && inner.startsWith(' ') && inner.endsWith(' ') && inner.trim()) {
    return inner.slice(1, -1);
  }
  return inner;
}

export function slugify(markdown: string): string {
  return markdown
    .replace(/`/g, '')
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-|-$/g, '');
}

function uniqueSlug(title: string, taken: Set<string>): string {
  const base = slugify(title) || 'entry';
  let slug = base;
  for (let n = 2; taken.has(slug); n++) slug = `${base}-${n}`;
  taken.add(slug);
  return slug;
}

