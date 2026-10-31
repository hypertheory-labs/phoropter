// The HTML is the API: what a person sees and what an agent fetches are the
// same pages. Structure is carried by rel links and microdata, whose names
// the profile page defines.

import { marked } from 'marked';
import type { Annotation, Draft, Prompt } from './draft.ts';
import type { Guide } from './guide.ts';
import type { Doc, EntryRef, Place, Plan, ResolvedAnchor, ResolvedEntry, Site } from './site.ts';
import type { Hunk, View } from './resolve.ts';

export type Page = { href: string; title: string; html: string };

export function renderSite(site: Site, guide: Guide): Page[] {
  const pages: Page[] = [home(site), profile(site), guidePage(site, guide)];
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
      const asBuilt = p.asBuilt ? `, with <a rel="as-built" itemprop="asBuilt" href="${rel(here, p.asBuilt.href)}">an as-built</a>` : '';
      const revisits = p.revisits ? `, revisits <a rel="revisits" itemprop="revisits" href="${rel(here, p.revisits.href)}">${esc(p.revisits.name)}</a>` : '';
      return `<li itemprop="plan" itemscope>
  <a rel="item" itemprop="url" href="${rel(here, p.href)}"><span itemprop="title">${inline(p.doc.title)}</span></a>
  <span class="meta"><span itemprop="name">${esc(p.name)}</span>, ${whenHtml(p)}, base ${baseHtml(p)}, ${entryCount(p.entries.length)}${asBuilt}${revisits}</span>
</li>`;
    })
    .join('\n');
  const places = site.places
    .map((pl) => `<li itemprop="place" itemscope>
  <a rel="item" itemprop="url" href="${rel(here, pl.href)}"><code itemprop="name">${esc(pl.label)}</code></a>
  <span class="meta"><span itemprop="kind">${pl.kind}</span>, ${entryCount(pl.refs.length)}</span>
</li>`)
    .join('\n');
  return layout(site, here, title, `
<h1>${esc(title)}</h1>
<div class="orientation">
<p>This is <strong>${esc(site.repoName)}</strong> read through its plans: the plans and their as-built files in <code>plans/</code>, each anchored to the code it was about, at commit <code>${esc(site.generated.short)}</code>. Nothing here is written to; it’s generated from git.</p>
<p>To find out why some code is the way it is, start from a file or symbol under <em>Anchored places</em>, which lists the plans that touch it, newest first. To read the history, start from <em>Plans</em>, oldest first. The <a rel="help" href="${rel(here, 'guide/')}">guide</a> says how plans are written, and the <a href="${rel(here, 'profile/')}">profile</a> defines every link relation and microdata property on these pages.</p>
</div>
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
<dt>Date</dt><dd>${whenHtml(plan)}</dd>
<dt>Base</dt><dd>${baseHtml(plan)}</dd>
<dt>File</dt><dd><code itemprop="file">${esc(plan.file)}</code></dd>
${asBuilt}
${revisitsHtml(here, plan)}
${frontmatterRest(plan.doc.frontmatter, ['cart', 'base', 'revisits'])}
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
  <span class="meta">in <span itemprop="plan">${esc(plan.name)}</span>, ${whenHtml(plan)}${doc === plan ? '' : ' (as-built)'}${anchor.isNew ? ', new<meta itemprop="new" content="true">' : ''}</span>
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
    ['help', 'registered', 'How are plans written? From every page to the guide, which is the same on every site.'],
    ['draft', 'phoropter', 'Is a plan being drafted? From every page to <code>draft/</code>, the repository’s one draft, while <code>plan.draft.md</code> exists. Only the dev server adds it; a built site never has a draft.'],
    ['item', 'registered', 'What does this collection hold? From the home page to each plan and anchored place.'],
    ['collection', 'registered', 'What is this a member of? From a plan or place to the home page.'],
    ['up', 'registered', 'What is this part of? From an as-built page to its plan.'],
    ['as-built', 'phoropter', 'What changed while this plan was being built? From a plan to its as-built file.'],
    ['anchored-at', 'phoropter', 'Where does this entry point? From an anchor on an entry to the page for its symbol or file, or to another plan’s entry, or to a whole plan.'],
    ['anchored-by', 'phoropter', 'Which plans point here? From a symbol or file page to each entry anchored there, and from an entry to each entry in another plan whose anchor names it.'],
    ['revisits', 'phoropter', 'Whose ground is this plan standing on? From a plan to the one plan its <code>revisits:</code> names (bodkin).'],
    ['revisited-by', 'phoropter', 'Has this plan’s ground been revisited? From a plan to each plan that revisits it.'],
    ['answers', 'phoropter', 'Which guess does this answer? From an as-built’s <code>Answered:</code> entry to the plan’s <code>Guess:</code> with the same title.'],
    ['answered-by', 'phoropter', 'Has this guess been answered? From a plan’s <code>Guess:</code> to the <code>Answered:</code> entry in its as-built.'],
  ];
  const props: [string, string][] = [
    ['(header)', 'Every page’s provenance, the item in its <code>&lt;header&gt;</code>: <code>repository</code> and <code>commit</code> (full sha in <code>value</code>).'],
    ['plan', 'A plan: <code>title</code>, <code>name</code> and <code>date</code> (from its filename), <code>added</code> (the commit that added it, which orders plans), <code>cart</code> as written, <code>base</code>, <code>file</code>, <code>intro</code>, <code>asBuilt</code>, <code>revisits</code> and <code>revisitedBy</code> links, and its <code>entry</code> items. On the home page, a plan has <code>entryCount</code> instead of its entries. When a <code>base</code> (or an as-built’s <code>head</code>) names a commit here, it’s a <code>&lt;data&gt;</code> whose value is the full sha. Otherwise its value is the text as written, like “none, a new repo”, or a commit a squash merge replaced; a missing base is “not given”.'],
    ['asBuilt', 'An as-built file: <code>title</code>, <code>plan</code>, <code>head</code>, <code>file</code>, and its <code>entry</code> items.'],
    ['entry', 'One entry: <code>kind</code> (Decision, Decline, Guess; in an as-built, Changed, Chose, Answered), <code>title</code>, <code>anchor</code> items, <code>reason</code>, <code>source</code> (the parenthesis after <em>Because</em> or <em>Answer</em>, exactly as written), <code>verdict</code> (an answer’s opening word: held, disproved, partly disproved, or not tested; on a guess, inside its <code>answeredBy</code> link’s sentence), <code>answers</code> or <code>answeredBy</code> links, and <code>anchoredBy</code> links from other plans’ entries.'],
    ['anchor', 'Where an entry points: <code>where</code> as written, <code>quote</code> if any, <code>new</code> (content <code>true</code>) for <code>where (new)</code>, in the same shape wherever it appears, <code>note</code> when it is shown but not resolved, <code>place</code> (a symbol or file page) or <code>target</code> (another plan’s entry, or a whole plan), and one <code>view</code> per commit. An anchor at an entry has no views unless it has a quote, and then one, at the generated commit: plans and landed as-builts don’t change.'],
    ['view', 'An anchor at one commit: <code>role</code> (base, as-built head, generated), <code>commit</code>, <code>status</code>, <code>file</code>, <code>startLine</code>/<code>endLine</code> for a symbol, <code>matches</code>, one <code>match</code> item per match with the <code>startLine</code> and <code>endLine</code> the quote covers, <code>note</code>, <code>code</code> (each one preceded by <code>codeStart</code>, the line number of its first line; the nth <code>codeStart</code> belongs to the nth <code>code</code>), and <code>sameAs</code> when a generated view matches the as-built head’s exactly and its code isn’t repeated. A plan’s anchors have up to three views: base, as-built head (where the plan’s own work landed), and generated.'],
    ['status', '<code>found</code>: the quote matched. <code>not-found</code>: it didn’t. <code>no-quote</code>: nothing to search for. <code>no-file</code>: the file isn’t at that commit. <code>no-symbol</code>: the symbol isn’t in that commit’s index. None of these judges a difference between commits.'],
    ['(draft header)', 'On the draft page, the header says it’s the working tree, not a commit: <code>file</code> (<code>plan.draft.md</code>), <code>repository</code>, <code>readAt</code>, the draft’s <code>sha256</code>, and the <code>commit</code> its anchors resolve at.'],
    ['draft', 'The draft page’s item: <code>title</code>, <code>base</code>, <code>revisits</code>, <code>notesSha256</code> and <code>annotates</code> (the draft version the notes were written against, when it isn’t this one), <code>intro</code>, <code>entry</code> items as on a plan (but never listed on places or other plans’ entries), <code>prompt</code> items, and <code>annotation</code> items.'],
    ['prompt', 'An open question in a draft, in the annotating agreement’s form: <code>kind</code> (Question, Select, Affirm), <code>text</code>, <code>option</code> for each of a Select’s choices, <code>fallback</code> (what happens if it goes unanswered), and <code>answer</code> when the notes restate it. It follows the entry (or intro) it belongs to.'],
    ['annotation', 'A note from <code>plan.draft.notes.md</code>: <code>heading</code> as written, <code>quote</code> (the draft’s source text it quotes), and <code>comment</code>. It follows the entry whose source holds its quote; a note without a quote, or whose quote isn’t in the draft anymore, comes first.'],
    ['place', 'A symbol or file that plans anchor to: <code>name</code>, <code>kind</code>, <code>symbol</code> or <code>file</code>, a symbol’s whole definition at the generated commit as <code>code</code> with its <code>codeStart</code>, <code>entryCount</code> on the home page, and <code>anchoredBy</code> items.'],
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

// The same guide on every site. Its own links are relative to `guide/`.
function guidePage(site: Site, guide: Guide): Page {
  const here = 'guide/';
  const html = block(guide.markdown);
  const title = /<h1[^>]*>([\s\S]*?)<\/h1>/.exec(html)?.[1] ?? 'Guide';
  const version = `<p class="meta">From phoropter, ${esc(guide.version)}. It describes the tool, so it’s the same on every site phoropter generates.</p>`;
  return layout(site, here, title, `
<nav><a href="${rel(here, '')}">${esc(site.repoName)} at ${esc(site.generated.short)}</a></nav>
<article class="guide">
${html.replace(/(<\/h1>)/, `$1\n${version}`)}
</article>`);
}

// The one draft, rendered per request by the dev server (loden). It says it's
// the working tree, not a commit; its anchors still resolve at commits.
export function draftPage(site: Site, draft: Draft): Page {
  const here = 'draft/';
  const { plan, notes } = draft;
  const at = draft.readAt.toISOString();
  const header = `Draft, not committed: <code itemprop="file">${esc(plan.file)}</code> in <span itemprop="repository">${esc(site.repoName)}</span>,
read at <time itemprop="readAt" datetime="${at}">${at.slice(11, 19)} UTC</time>
<span class="meta">sha256 <data itemprop="sha256" value="${draft.sha256}">${draft.sha256.slice(0, 12)}</data>; anchors at <data itemprop="commit" value="${site.generated.sha}">${esc(site.generated.short)}</data></span>`;
  const byPlace = (key: string) => (notes?.annotations ?? []).filter((a) => a.found && a.at === key);
  const top = (notes?.annotations ?? []).filter((a) => a.quote === undefined || !a.found);
  const extras = (key: string) => (draft.prompts.get(key) ?? []).map(promptHtml).join('\n') + byPlace(key).map(annotationHtml).join('\n');
  const notesFact = notes
    ? `<dt>Notes</dt><dd><code>plan.draft.notes.md</code>, sha256 <data itemprop="notesSha256" value="${notes.sha256}">${notes.sha256.slice(0, 12)}</data>${notesAnnotate(draft)}</dd>`
    : '<dt>Notes</dt><dd class="note">none yet</dd>';
  return layout(site, here, plan.doc.title || 'Draft', `
<nav><a href="${rel(here, '')}">${esc(site.repoName)} at ${esc(site.generated.short)}</a></nav>
<article class="draft" itemscope>
<p class="draft-label">Draft. Nothing here is testimony yet.</p>
<h1 itemprop="title">${inline(plan.doc.title || 'Untitled draft')}</h1>
<dl class="facts">
<dt>Base</dt><dd>${baseHtml(plan)}</dd>
${revisitsHtml(here, plan)}
${notesFact}
</dl>
${top.map(annotationHtml).join('\n')}
${plan.doc.intro ? `<div class="intro" itemprop="intro">${block(plan.doc.intro)}</div>` : ''}
${extras('intro')}
${plan.entries.map((e) => entryHtml(here, plan, e) + extras(e.slug)).join('\n')}
</article>`, header);
}

// Notes say which version of the draft they annotate. When it isn't this one,
// their quotes may have come loose.
function notesAnnotate(draft: Draft): string {
  const annotates = draft.notes?.annotates;
  if (!annotates) return '';
  return draft.notes!.current
    ? `, annotating this version`
    : `, annotating <data itemprop="annotates" value="${annotates}">${annotates}</data> <span class="note">an earlier version of the draft; quotes may have come loose</span>`;
}

function promptHtml(p: Prompt): string {
  const options = p.options.length ? `<ul>${p.options.map((o) => `<li itemprop="option">${inline(o)}</li>`).join('')}</ul>` : '';
  const fallback = p.fallback
    ? `<p class="fallback"><strong>If unanswered:</strong> <span itemprop="fallback">${inline(p.fallback)}</span></p>`
    : '<p class="fallback note">No fallback given, so silence doesn’t say what happens.</p>';
  const answer = p.answer
    ? `<div class="answer" itemprop="answer">${block(p.answer.comment)}</div>`
    : '<p class="note">Not answered yet.</p>';
  return `<div class="prompt" itemprop="prompt" itemscope>
<p class="prompt-heading"><span class="kind" itemprop="kind">${esc(p.kind)}</span>: <span itemprop="text">${inline(p.text)}</span></p>
${p.body ? block(p.body) : ''}${options}
${fallback}
${answer}
</div>`;
}

function annotationHtml(a: Annotation): string {
  const missing = a.quote !== undefined && !a.found ? '<p class="note">quote not found in the draft: it moved under the note</p>' : '';
  return `<aside class="annotation" itemprop="annotation" itemscope>
<p class="meta" itemprop="heading">${esc(a.heading)}</p>
${a.quote !== undefined ? `<blockquote itemprop="quote">${esc(a.quote)}</blockquote>` : ''}${missing}
${a.comment ? `<div itemprop="comment">${block(a.comment)}</div>` : ''}
</aside>`;
}

// --- parts ---

function entryHtml(here: string, doc: Doc, entry: ResolvedEntry): string {
  const heading = entry.kind
    ? `<span class="kind kind-${entry.kind.toLowerCase()}" itemprop="kind">${esc(entry.kind)}</span>: <span itemprop="title">${inline(entry.title)}</span>`
    : `<span itemprop="title">${inline(entry.title)}</span>`;
  return `<section class="entry" id="${entry.slug}" itemprop="entry" itemscope>
<h2><a class="self" href="#${entry.slug}">${heading}</a></h2>
${pairHtml(here, entry)}
${anchoredByHtml(here, entry)}
${entry.resolved.map((a) => anchorHtml(here, doc, a)).join('\n')}
${entry.body ? `<div class="reason" itemprop="reason">${withSource(block(entry.body))}</div>` : ''}
</section>`;
}

// The source stays as written, inside the bold lead where the reader sees it;
// the span only makes it findable.
function withSource(html: string): string {
  return html.replace(/^<p><strong>(Because|Answer) \((.*?)\):<\/strong>/, '<p><strong>$1 (<span itemprop="source">$2</span>):</strong>');
}

function pairHtml(here: string, entry: ResolvedEntry): string {
  const link = (r: string, prop: string, to: { doc: Doc; entry: ResolvedEntry }, text: string) =>
    `<a rel="${r}" itemprop="${prop}" href="${rel(here, to.doc.href)}#${to.entry.slug}">${text}</a>`;
  if (entry.kind === 'Answered') {
    const verdict = verdictHtml(entry);
    if (entry.answers) return `<p class="pair">Answers ${link('answers', 'answers', entry.answers, 'the plan’s guess')}: ${verdict}.</p>`;
    return `<p class="pair">${verdict}. <span class="note">No guess in the plan has this title.</span></p>`;
  }
  if (entry.answeredBy.length) {
    return `<p class="pair">Answered in ${entry.answeredBy.map((a) => `${link('answered-by', 'answeredBy', a, 'the as-built')}: ${verdictHtml(a.entry)}`).join(', ')}.</p>`;
  }
  return '';
}

// Grommet's opening word, or a plain statement that there isn't one. The
// qualifiers after it stay in the prose.
function verdictHtml(answer: ResolvedEntry): string {
  return answer.verdict
    ? `<strong class="verdict verdict-${answer.verdict.replace(' ', '-')}" itemprop="verdict">${answer.verdict}</strong>`
    : '<span class="note">no verdict (it doesn’t open with held, disproved, partly disproved, or not tested)</span>';
}

function anchoredByHtml(here: string, entry: ResolvedEntry): string {
  if (!entry.anchoredBy.length) return '';
  const items = entry.anchoredBy.map(
    (r) => `<a rel="anchored-by" itemprop="anchoredBy" href="${rel(here, r.doc.href)}#${r.entry.slug}">${docName(r)}, ${kindTitle(r.entry)}</a>`,
  );
  return `<p class="pair">Anchored by ${items.join('; ')}.</p>`;
}

function revisitsHtml(here: string, plan: Plan): string {
  const text = plan.doc.frontmatter.revisits;
  const out: string[] = [];
  if (plan.revisits) {
    out.push(`<dt>Revisits</dt><dd><a rel="revisits" itemprop="revisits" href="${rel(here, plan.revisits.href)}">${inline(plan.revisits.doc.title)}</a> <span class="meta">${esc(plan.revisits.name)}</span></dd>`);
  } else if (text !== undefined) {
    out.push(`<dt>Revisits</dt><dd><code itemprop="revisits">${esc(text)}</code> <span class="note">not a plan on this site</span></dd>`);
  }
  if (plan.revisitedBy.length) {
    const links = plan.revisitedBy.map((p) => `<a rel="revisited-by" itemprop="revisitedBy" href="${rel(here, p.href)}">${inline(p.doc.title)}</a> <span class="meta">${esc(p.name)}</span>`);
    out.push(`<dt>Revisited by</dt><dd>${links.join('<br>')}</dd>`);
  }
  return out.join('\n');
}

function docName(r: EntryRef): string {
  return esc(r.doc === r.plan ? r.plan.name : `${r.plan.name}'s as-built`);
}

function kindTitle(entry: ResolvedEntry): string {
  return entry.kind ? `${esc(entry.kind)}: ${inline(entry.title)}` : inline(entry.title);
}

function anchorHtml(here: string, doc: Doc, ra: ResolvedAnchor): string {
  const { anchor } = ra;
  const label = anchor.isNew ? 'where (new)' : 'where';
  const place = ra.place
    ? ` <a rel="anchored-at" itemprop="place" href="${rel(here, ra.place.href)}">${ra.place.kind} page</a>`
    : '';
  const target = ra.entry
    ? `<p class="target"><span class="label">entry</span> <a rel="anchored-at" itemprop="target" href="${rel(here, ra.entry.doc.href)}#${ra.entry.entry.slug}">${kindTitle(ra.entry.entry)}</a> <span class="meta">in ${docName(ra.entry)}</span></p>`
    : ra.plan
      ? `<p class="target"><span class="label">plan</span> <a rel="anchored-at" itemprop="target" href="${rel(here, ra.plan.href)}">${inline(ra.plan.doc.title)}</a> <span class="meta">${esc(ra.plan.name)}</span></p>`
      : '';
  const quote = anchor.quote !== undefined ? `<p class="quote"><span class="label">quote</span> <code itemprop="quote">${esc(anchor.quote)}</code></p>` : '';
  const note = ra.note ? `<p class="note" itemprop="note">${esc(ra.note)}</p>` : '';
  const views = ra.views.length ? `<div class="views">${ra.views.map(viewHtml).join('\n')}</div>` : '';
  return `<div class="anchor" itemprop="anchor" itemscope>
${anchor.isNew ? '<meta itemprop="new" content="true">' : ''}
<p class="where"><span class="label">${label}</span> <code itemprop="where">${esc(anchor.where)}</code>${place}</p>
${target}${quote}${note}${views}
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
<meta itemprop="matches" content="${v.matches}">${v.at.map(matchHtml).join('')}${range}${v.sameAs ? `<meta itemprop="sameAs" content="${v.sameAs}">` : ''}
<span class="note" itemprop="note">${inline(v.note)}</span>
</figcaption>
${code}
</figure>`;
}

// One item per match, so a reader finds the quote's lines without the note.
function matchHtml(lines: number[]): string {
  return `<span itemprop="match" itemscope hidden><meta itemprop="startLine" content="${lines[0]}"><meta itemprop="endLine" content="${lines[lines.length - 1]}"></span>`;
}

// codeStart comes just before the code it numbers; microdata keeps document
// order, so the nth codeStart belongs to the nth code.
function hunkHtml(h: Hunk): string {
  const marked = new Set(h.marked);
  const lines = h.lines.map((line, i) => {
    const n = h.start + i;
    return marked.has(n) ? `<mark class="line">${esc(line)}</mark>` : `<span class="line">${esc(line)}</span>`;
  });
  // The numbers are a CSS counter from codeStart, so code's text is just the code.
  return `<meta itemprop="codeStart" content="${h.start}"><pre itemprop="code" data-start="${h.start}"><code style="counter-reset: line ${h.start - 1}">${lines.join('\n')}</code></pre>`;
}

function layout(site: Site, here: string, title: string, main: string, header?: string): Page {
  // The home page's title already is the provenance, so it isn't said twice.
  const provenance = `${site.repoName} at ${site.generated.short}`;
  header ??= `Generated from <span itemprop="repository">${esc(site.repoName)}</span>
at <data itemprop="commit" value="${site.generated.sha}"><code>${esc(site.generated.short)}</code></data>
<span class="meta">${site.generated.sha}</span>`;
  const html = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>${esc(here ? `${stripMarkdown(title)} · ${provenance}` : provenance)}</title>
<link rel="profile" href="${rel(here, 'profile/')}">
<link rel="help" href="${rel(here, 'guide/')}">
<link rel="stylesheet" href="${rel(here, 'style.css')}">
</head>
<body>
<header class="provenance" itemscope>
${header}
<span class="links"><a class="home-link" href="${rel(here, '')}">home</a>
<a rel="help" href="${rel(here, 'guide/')}">guide</a>
<a class="profile-link" href="${rel(here, 'profile/')}">profile</a></span>
</header>
<main>
${main}
</main>
</body>
</html>
`;
  return { href: here, title, html };
}

// The filename's date, and the commit that added the plan, which is what
// orders plans that share a date.
function whenHtml(plan: Plan): string {
  const date = plan.date ? `<time itemprop="date" datetime="${plan.date}">${plan.date}</time>` : 'no date';
  if (!plan.added) return date;
  const at = new Date(plan.added.time * 1000).toISOString();
  return `${date}, added in <data itemprop="added" value="${plan.added.sha}"><code>${esc(plan.added.short)}</code></data> <time datetime="${at}">${at.slice(11, 16)} UTC</time>`;
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

function entryCount(n: number): string {
  return `<data itemprop="entryCount" value="${n}">${count(n, 'entry', 'entries')}</data>`;
}

function count(n: number, one: string, many: string): string {
  return `${n} ${n === 1 ? one : many}`;
}

function esc(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}
