// node src/serve.ts <repo> [commit] [--out site] [--port 4173], or `phoropter`.
// Builds the site once, then serves it read-only. A commit's site never
// changes, so there's nothing to watch. The exceptions are in the repo's
// working tree, under drafts/: the draft, read on every request to draft/
// (loden), and linked from every page's header while it exists; and notes on
// landed plans, shown on each plan's page and counted on the home page
// (tarlatan, organza).

import { readFileSync, statSync } from 'node:fs';
import { createServer } from 'node:http';
import { extname, join, normalize, resolve } from 'node:path';
import { build, cli } from './build.ts';
import type { Site } from './site.ts';
import type { Store } from './store.ts';
import { DRAFT, hasDraft, readDraft, readPlanNotes } from './draft.ts';
import { annotationHtml, draftPage, rel } from './render.ts';

const types: Record<string, string> = {
  '.html': 'text/html; charset=utf-8',
  '.css': 'text/css; charset=utf-8',
};

// One server per process, so what the handler reads is set once, by serve().
let site: Site;
let root: string;
let repoDir: string;

export type Served = { port: number; url: string; title: string; pages: number; site: Site; close: () => void };

// Builds once, then serves. A taken port moves to the next free one, unless
// it was asked for exactly (batik).
export function serve(opts: { repo: string; ref: string; out: string; port: number; exactPort?: boolean; store?: Store }): Promise<Served> {
  const built = build(opts.repo, opts.ref, opts.out, opts.store);
  site = built.site;
  root = resolve(opts.out);
  repoDir = resolve(opts.repo);
  const server = createServer(handle);
  return new Promise((ok, fail) => {
    let port = opts.port;
    server.on('error', (err: NodeJS.ErrnoException) => {
      if (err.code === 'EADDRINUSE' && !opts.exactPort && port < opts.port + 100) server.listen(++port, '127.0.0.1');
      else fail(err);
    });
    server.on('listening', () => ok({ port, url: `http://localhost:${port}/`, title: built.title, pages: built.pages, site: built.site, close: () => server.close() }));
    server.listen(port, '127.0.0.1');
  });
}

if (import.meta.main) {
  const args = cli(process.argv.slice(2));
  const served = await serve({ ...args, exactPort: true });
  console.log(`${served.title}: ${served.pages} pages, serving ${served.url}`);
}

function handle(req: import('node:http').IncomingMessage, res: import('node:http').ServerResponse): void {
  if (req.method !== 'GET' && req.method !== 'HEAD') {
    res.writeHead(405, { allow: 'GET, HEAD' }).end();
    return;
  }
  const path = decodeURIComponent(new URL(req.url ?? '/', 'http://localhost').pathname);
  if (path === '/draft') {
    res.writeHead(301, { location: '/draft/' }).end();
    return;
  }
  if (path === '/draft/') {
    let draft;
    try {
      draft = readDraft(site, repoDir);
    } catch (err) {
      res.writeHead(500, { 'content-type': 'text/plain; charset=utf-8' }).end(`couldn't read ${DRAFT}: ${(err as Error).message}\n`);
      return;
    }
    if (!draft) {
      res.writeHead(404, { 'content-type': 'text/plain; charset=utf-8' }).end(`no ${DRAFT} in ${repoDir}\n`);
      return;
    }
    send(res, req.method, 'text/html; charset=utf-8', withDraftLink(draftPage(site, draft).html, 'draft/'));
    return;
  }
  let file = normalize(join(root, path));
  if (!file.startsWith(root)) {
    res.writeHead(403).end();
    return;
  }
  try {
    if (statSync(file).isDirectory()) {
      if (!path.endsWith('/')) {
        res.writeHead(301, { location: encodeURI(path) + '/' }).end();
        return;
      }
      file = join(file, 'index.html');
    }
    statSync(file);
  } catch {
    res.writeHead(404, { 'content-type': 'text/plain' }).end('not found\n');
    return;
  }
  const type = types[extname(file)] ?? 'application/octet-stream';
  const body = readFileSync(file);
  const here = path.slice(1).replace(/index\.html$/, '');
  send(res, req.method, type, type.startsWith('text/html') ? withDraftLink(withPlanNotes(body.toString('utf8'), here), path.slice(1)) : body);
}

function send(res: import('node:http').ServerResponse, method: string | undefined, type: string, body: string | Buffer): void {
  res.writeHead(200, { 'content-type': type, 'cache-control': 'no-store' });
  res.end(method === 'HEAD' ? undefined : body);
}

// Built pages don't know about notes on landed plans. On a plan's page, they go
// after the entries they quote, and notes on the whole plan before the first
// entry. On the home page, each plan with notes says how many.
function withPlanNotes(html: string, here: string): string {
  if (here === '') {
    for (const plan of site.plans) {
      const notes = readPlanNotes(repoDir, plan);
      if (!notes) continue;
      const start = html.indexOf(`itemprop="url" href="${plan.href}"`);
      const end = start === -1 ? -1 : html.indexOf('</span>\n</li>', start);
      if (end === -1) continue;
      const n = notes.length;
      html = html.slice(0, end) + `, <data itemprop="noteCount" value="${n}">${n === 1 ? '1 note' : `${n} notes`}</data>` + html.slice(end);
    }
    return html;
  }
  const plan = site.plans.find((p) => p.href === here);
  const notes = plan && readPlanNotes(repoDir, plan);
  if (!plan || !notes) return html;
  for (const e of plan.entries) {
    const mine = notes.filter((a) => a.found && a.at === e.slug);
    const start = html.indexOf(`<section class="entry" id="${e.slug}"`);
    const end = start === -1 ? -1 : html.indexOf('</section>', start);
    if (!mine.length || end === -1) continue;
    const at = end + '</section>'.length;
    html = html.slice(0, at) + '\n' + mine.map(annotationHtml).join('\n') + html.slice(at);
  }
  const top = notes.filter((a) => a.quote === undefined || !a.found || a.at === 'intro');
  const first = html.indexOf('<section class="entry"');
  if (first !== -1) {
    const label = `<p class="note">Notes on this plan, from <code>drafts/</code> in the working tree. The plan itself stays as written.</p>\n`;
    html = html.slice(0, first) + label + top.map(annotationHtml).join('\n') + '\n' + html.slice(first);
  }
  return html;
}

// Built pages don't know about the draft; the server adds the link while one
// exists, relative to the page it's on.
function withDraftLink(html: string, here: string): string {
  if (!hasDraft(repoDir)) return html;
  const dir = here.endsWith('/') || here === '' ? here : here.slice(0, here.lastIndexOf('/') + 1);
  return html.replace(/(<a class="home-link"[^>]*>home<\/a>)/, `$1\n<a rel="draft" href="${rel(dir, 'draft/')}">draft</a>`);
}
