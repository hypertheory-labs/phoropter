// node src/serve.ts <repo> [commit] [--out site] [--port 4173]
// Builds the site once, then serves it read-only. A commit's site never
// changes, so there's nothing to watch. The one exception is the draft,
// `plan.draft.md` in the repo's working tree, which is read on every request
// to draft/ (loden); while it exists, every page's header links it.

import { readFileSync, statSync } from 'node:fs';
import { createServer } from 'node:http';
import { extname, join, normalize, resolve } from 'node:path';
import { build, cli } from './build.ts';
import { DRAFT, hasDraft, readDraft } from './draft.ts';
import { draftPage, rel } from './render.ts';

const types: Record<string, string> = {
  '.html': 'text/html; charset=utf-8',
  '.css': 'text/css; charset=utf-8',
};

const args = cli(process.argv.slice(2));
const { pages, title, site } = build(args.repo, args.ref, args.out);
const root = resolve(args.out);
const repoDir = resolve(args.repo);

createServer((req, res) => {
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
  send(res, req.method, type, type.startsWith('text/html') ? withDraftLink(body.toString('utf8'), path.slice(1)) : body);
}).listen(args.port, '127.0.0.1', () => {
  console.log(`${title}: ${pages} pages, serving http://localhost:${args.port}/`);
});

function send(res: import('node:http').ServerResponse, method: string | undefined, type: string, body: string | Buffer): void {
  res.writeHead(200, { 'content-type': type, 'cache-control': 'no-store' });
  res.end(method === 'HEAD' ? undefined : body);
}

// Built pages don't know about the draft; the server adds the link while one
// exists, relative to the page it's on.
function withDraftLink(html: string, here: string): string {
  if (!hasDraft(repoDir)) return html;
  const dir = here.endsWith('/') || here === '' ? here : here.slice(0, here.lastIndexOf('/') + 1);
  return html.replace('<span class="links">', `<span class="links"><a rel="draft" href="${rel(dir, 'draft/')}">draft</a>\n`);
}
