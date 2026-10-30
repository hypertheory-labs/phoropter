// The HTML is the API: what a person sees and what an agent fetches are the
// same pages. Structure is carried by rel links and microdata, whose names
// the profile page defines.

import { marked } from 'marked';
import type { Doc, Place, Plan, ResolvedAnchor, ResolvedEntry, Site } from './site.ts';
import type { Hunk, View } from './resolve.ts';

export type Page = { href: string; title: string; html: string };

export function renderSite(site: Site): Page[] {
  const pages: Page[] = [home(site), profile(site)];
  for (const plan of site.plans) {
    pages.push(planPage(site, plan));
    if (plan.asBuilt) pages.push(asBuiltPage(site, plan, plan.asBuilt));
  }
  for (const place of site.places) pages.push(placePage(site, place));
  return pages;
}

// --- pages ---

function home(site: Site): Page {
  const here = '';
  const title = `${site.repoName} at ${site.generated.short}`;
  const plans = site.plans
    .map((p) => {
      const asBuilt = p.asBuilt ? `, with <a href="${rel(here, p.asBuilt.href)}">an as-built</a>` : '';
      return `<li itemprop="plan" itemscope>
  <a rel="item" itemprop="url" href="${rel(here, p.href)}"><span itemprop="title">${inline(p.doc.title)}</span></a>
  <span class="meta"><span itemprop="name">${esc(p.name)}</span>, base ${baseHtml(p)}, ${count(p.entries.length, 'entry', 'entries')}${asBuilt}</span>
</li>`;
    })
    .join('\n');
  const places = site.places
    .map((pl) => `<li itemprop="place" itemscope>
  <a rel="item" itemprop="url" href="${rel(here, pl.href)}"><code itemprop="name">${esc(pl.label)}</code></a>
  <span class="meta"><span itemprop="kind">${pl.kind}</span>, ${count(pl.refs.length, 'entry', 'entries')}</span>
</li>`)
    .join('\n');
  return layout(site, here, title, `
<h1>${esc(title)}</h1>
<div itemscope>
<section>
<h2>Plans</h2>
${plans ? `<ul class="list">${plans}</ul>` : '<p>No plans in <code>plans/</code> at this commit.</p>'}
</section>
<section>
<h2>Anchored places</h2>
<p>Each symbol or file a plan points at has a page listing the plans there.</p>
${places ? `<ul class="list">${places}</ul>` : '<p>Nothing is anchored.</p>'}
</section>
</div>`);
}

function planPage(site: Site, plan: Plan): Page {
  const here = plan.href;
  const asBuilt = plan.asBuilt
    ? `<dt>As built</dt><dd><a rel="as-built" itemprop="asBuilt" href="${rel(here, plan.asBuilt.href)}">${esc(plan.asBuilt.file)}</a></dd>`
    : '';
  return layout(site, here, plan.doc.title, `
<nav><a rel="collection" href="${rel(here, '')}">${esc(site.repoName)} at ${esc(site.generated.short)}</a></nav>
<article itemscope>
<h1 itemprop="title">${inline(plan.doc.title)}</h1>
<dl class="facts">
<dt>Name</dt><dd itemprop="name">${esc(plan.name)}</dd>
${plan.doc.frontmatter.cart !== undefined ? `<dt>Cart</dt><dd itemprop="cart">${esc(plan.doc.frontmatter.cart)}</dd>` : ''}
<dt>Base</dt><dd>${baseHtml(plan)}</dd>
<dt>File</dt><dd><code itemprop="file">${esc(plan.file)}</code></dd>
${asBuilt}
${frontmatterRest(plan.doc.frontmatter, ['cart', 'base'])}
</dl>
${plan.doc.intro ? `<div class="intro" itemprop="intro">${block(plan.doc.intro)}</div>` : ''}
${plan.entries.map((e) => entryHtml(here, plan, e)).join('\n')}
</article>`);
}

function asBuiltPage(site: Site, plan: Plan, doc: Doc): Page {
  const here = doc.href;
  const head = doc.before
    ? `<dt>Head</dt><dd><data itemprop="head" value="${doc.before.sha}"><code>${esc(doc.before.short)}</code></data></dd>`
    : doc.beforeText
      ? `<dt>Head</dt><dd itemprop="head">${esc(doc.beforeText)}</dd>`
      : '';
  return layout(site, here, doc.doc.title, `
<nav><a rel="up" href="${rel(here, plan.href)}">${inline(plan.doc.title)}</a></nav>
<article itemscope>
<h1 itemprop="title">${inline(doc.doc.title)}</h1>
<dl class="facts">
<dt>Plan</dt><dd><a rel="up" itemprop="plan" href="${rel(here, plan.href)}">${esc(plan.file)}</a></dd>
${head}
<dt>File</dt><dd><code itemprop="file">${esc(doc.file)}</code></dd>
${frontmatterRest(doc.doc.frontmatter, ['head'])}
</dl>
${doc.doc.intro ? `<div class="intro" itemprop="intro">${block(doc.doc.intro)}</div>` : ''}
${doc.entries.map((e) => entryHtml(here, doc, e)).join('\n')}
</article>`);
}

function placePage(site: Site, place: Place): Page {
  const here = place.href;
  const def = place.definition;
  const location = def
    ? `<dt>At ${esc(site.generated.short)}</dt><dd><code itemprop="file">${esc(def.file)}</code>, lines <span itemprop="startLine">${def.startLine}</span>–<span itemprop="endLine">${def.endLine}</span></dd>`
    : place.kind === 'symbol'
      ? `<dt>At ${esc(site.generated.short)}</dt><dd>not in the index</dd>`
      : '';
  const refs = place.refs
    .map(({ plan, doc, entry, anchor }) => `<li itemprop="anchoredBy" itemscope>
  <a rel="anchored-by" itemprop="url" href="${rel(here, doc.href)}#${entry.slug}">${entry.kind ? `<span itemprop="kind">${esc(entry.kind)}</span>: ` : ''}<span itemprop="title">${inline(entry.title)}</span></a>
  <span class="meta">in <span itemprop="plan">${esc(plan.name)}</span>${doc === plan ? '' : ' (as-built)'}${anchor.isNew ? ', <span itemprop="new">new</span>' : ''}</span>
  <code class="where" itemprop="where">${esc(anchor.where)}</code>
</li>`)
    .join('\n');
  return layout(site, here, place.label, `
<nav><a rel="collection" href="${rel(here, '')}">${esc(site.repoName)} at ${esc(site.generated.short)}</a></nav>
<article itemscope>
<h1><code itemprop="name">${esc(place.label)}</code></h1>
<dl class="facts">
<dt>Kind</dt><dd itemprop="kind">${place.kind}</dd>
<dt>${place.kind === 'symbol' ? 'Symbol' : 'Path'}</dt><dd><code itemprop="${place.kind}">${esc(place.full)}</code></dd>
${location}
</dl>
${place.code ? `<div class="view">${hunkHtml(place.code)}</div>` : ''}
<h2>Plans anchored here</h2>
<ul class="list">${refs}</ul>
</article>`);
}

function profile(site: Site): Page {
  const here = 'profile/';
  const rels: [string, string, string][] = [
    ['profile', 'registered', 'What do the rels and properties on this page mean? This document.'],
    ['item', 'registered', 'What does this collection hold? From the home page to each plan and anchored place.'],
    ['collection', 'registered', 'What is this a member of? From a plan or place to the home page.'],
    ['up', 'registered', 'What is this part of? From an as-built page to its plan.'],
    ['as-built', 'phoropter', 'What changed while this plan was being built? From a plan to its as-built file.'],
    ['anchored-at', 'phoropter', 'Where in the code does this entry point? From an anchor on an entry to the page for its symbol or file.'],
    ['anchored-by', 'phoropter', 'Which plans point here? From a symbol or file page to each entry anchored there.'],
  ];
  const props: [string, string][] = [
    ['(header)', 'Every page’s provenance, the item in its <code>&lt;header&gt;</code>: <code>repository</code> and <code>commit</code> (full sha in <code>value</code>).'],
    ['plan', 'A plan: <code>title</code>, <code>name</code> (from its filename), <code>cart</code> as written, <code>base</code>, <code>file</code>, <code>intro</code>, <code>asBuilt</code>, and its <code>entry</code> items.'],
    ['asBuilt', 'An as-built file: <code>title</code>, <code>plan</code>, <code>head</code>, <code>file</code>, and its <code>entry</code> items.'],
    ['entry', 'One entry: <code>kind</code> (Decision, Decline, Guess, Changed), <code>title</code>, <code>anchor</code> items, and <code>reason</code>.'],
    ['anchor', 'Where an entry points: <code>where</code> as written, <code>quote</code> if any, <code>new</code> for <code>where (new)</code>, <code>note</code> when it is shown but not resolved, and one <code>view</code> per commit.'],
    ['view', 'An anchor at one commit: <code>role</code> (base, as-built head, generated), <code>commit</code>, <code>status</code>, <code>file</code>, <code>startLine</code>/<code>endLine</code> for a symbol, <code>matches</code>, <code>note</code>, and <code>code</code>.'],
    ['status', '<code>found</code>: the quote matched. <code>not-found</code>: it didn’t. <code>no-quote</code>: nothing to search for. <code>no-file</code>: the file isn’t at that commit. <code>no-symbol</code>: the symbol isn’t in that commit’s index. None of these judges a difference between commits.'],
    ['place', 'A symbol or file that plans anchor to: <code>name</code>, <code>kind</code>, <code>symbol</code> or <code>file</code>, a symbol’s whole definition at the generated commit as <code>code</code>, and <code>anchoredBy</code> items.'],
  ];
  return layout(site, here, 'Profile', `
<nav><a href="${rel(here, '')}">${esc(site.repoName)} at ${esc(site.generated.short)}</a></nav>
<h1>Profile</h1>
<p>Phoropter’s pages are its API. A person and an agent read the same HTML; structure is in <code>rel</code> links and microdata (items have no <code>itemtype</code>; property names are defined here). The top-level item in each page’s <code>&lt;main&gt;</code> is the plan, as-built, place, or home listing the page is about.</p>
<h2>Link relations</h2>
<dl class="defs">
${rels.map(([name, source, q]) => `<dt id="${name}"><code>${name}</code> <span class="meta">${source}</span></dt><dd>${q}</dd>`).join('\n')}
</dl>
<h2>Microdata properties</h2>
<dl class="defs">
${props.map(([name, d]) => `<dt id="prop-${name}"><code>${name}</code></dt><dd>${d}</dd>`).join('\n')}
</dl>`);
}

// --- parts ---

function entryHtml(here: string, doc: Doc, entry: ResolvedEntry): string {
  const heading = entry.kind
    ? `<span class="kind kind-${entry.kind.toLowerCase()}" itemprop="kind">${esc(entry.kind)}</span>: <span itemprop="title">${inline(entry.title)}</span>`
    : `<span itemprop="title">${inline(entry.title)}</span>`;
  return `<section class="entry" id="${entry.slug}" itemprop="entry" itemscope>
<h2><a class="self" href="#${entry.slug}">${heading}</a></h2>
${entry.resolved.map((a) => anchorHtml(here, doc, a)).join('\n')}
${entry.body ? `<div class="reason" itemprop="reason">${block(entry.body)}</div>` : ''}
</section>`;
}

function anchorHtml(here: string, doc: Doc, ra: ResolvedAnchor): string {
  const { anchor } = ra;
  const label = anchor.isNew ? 'where (new)' : 'where';
  const place = ra.place
    ? ` <a rel="anchored-at" itemprop="place" href="${rel(here, ra.place.href)}">${ra.place.kind} page</a>`
    : '';
  const quote = anchor.quote !== undefined ? `<p class="quote"><span class="label">quote</span> <code itemprop="quote">${esc(anchor.quote)}</code></p>` : '';
  const note = ra.note ? `<p class="note" itemprop="note">${esc(ra.note)}</p>` : '';
  const views = ra.views.length ? `<div class="views">${ra.views.map(viewHtml).join('\n')}</div>` : '';
  return `<div class="anchor" itemprop="anchor" itemscope>
${anchor.isNew ? '<meta itemprop="new" content="true">' : ''}
<p class="where"><span class="label">${label}</span> <code itemprop="where">${esc(anchor.where)}</code>${place}</p>
${quote}${note}${views}
</div>`;
}

function viewHtml(v: View): string {
  const range = v.startLine ? `<meta itemprop="startLine" content="${v.startLine}"><meta itemprop="endLine" content="${v.endLine}">` : '';
  const code = v.hunks.map(hunkHtml).join('\n');
  return `<figure class="view status-${v.status}" itemprop="view" itemscope>
<figcaption>
<span class="role" itemprop="role">${esc(v.role)}</span>
<data itemprop="commit" value="${v.sha}"><code>${esc(v.short)}</code></data>
<span class="status" itemprop="status">${v.status}</span>
${v.file ? `<code class="file" itemprop="file">${esc(v.file)}</code>` : ''}
<meta itemprop="matches" content="${v.matches}">${range}
<span class="note" itemprop="note">${inline(v.note)}</span>
</figcaption>
${code}
</figure>`;
}

function hunkHtml(h: Hunk): string {
  const marked = new Set(h.marked);
  const lines = h.lines.map((line, i) => {
    const n = h.start + i;
    const text = `<span class="ln">${n}</span>${esc(line) || ' '}`;
    return marked.has(n) ? `<mark>${text}</mark>` : text;
  });
  return `<pre itemprop="code" data-start="${h.start}"><code>${lines.join('\n')}</code></pre>`;
}

function layout(site: Site, here: string, title: string, main: string): Page {
  // The home page's title already is the provenance, so it isn't said twice.
  const provenance = `${site.repoName} at ${site.generated.short}`;
  const html = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>${esc(here ? `${stripMarkdown(title)} · ${provenance}` : provenance)}</title>
<link rel="profile" href="${rel(here, 'profile/')}">
<link rel="stylesheet" href="${rel(here, 'style.css')}">
</head>
<body>
<header class="provenance" itemscope>
Generated from <span itemprop="repository">${esc(site.repoName)}</span>
at <data itemprop="commit" value="${site.generated.sha}"><code>${esc(site.generated.short)}</code></data>
<span class="meta">${site.generated.sha}</span>
</header>
<main>
${main}
</main>
</body>
</html>
`;
  return { href: here, title, html };
}

function baseHtml(plan: Plan): string {
  if (plan.before) return `<data itemprop="base" value="${plan.before.sha}"><code>${esc(plan.before.short)}</code></data>`;
  return `<span itemprop="base">${esc(plan.beforeText || 'not given')}</span>`;
}

function frontmatterRest(fm: Record<string, string>, shown: string[]): string {
  return Object.entries(fm)
    .filter(([k]) => !shown.includes(k))
    .map(([k, v]) => `<dt>${esc(k[0].toUpperCase() + k.slice(1))}</dt><dd>${esc(v)}</dd>`)
    .join('\n');
}

// A relative href from one site directory to another, so the site works from
// any root, including file://.
export function rel(from: string, to: string): string {
  const depth = from.split('/').filter(Boolean).length;
  const path = '../'.repeat(depth) + encodeURI(to);
  return path || './';
}

function inline(md: string): string {
  return marked.parseInline(md, { async: false });
}

function block(md: string): string {
  return marked.parse(md, { async: false });
}

function stripMarkdown(md: string): string {
  return md.replace(/`/g, '');
}

function count(n: number, one: string, many: string): string {
  return `${n} ${n === 1 ? one : many}`;
}

function esc(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}
