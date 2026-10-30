// node src/serve.ts <repo> [commit] [--out site] [--port 4173]
// Builds the site once, then serves it read-only. A commit's site never
// changes, so there's nothing to watch.

import { createReadStream, statSync } from 'node:fs';
import { createServer } from 'node:http';
import { extname, join, normalize, resolve } from 'node:path';
import { build, cli } from './build.ts';

const types: Record<string, string> = {
  '.html': 'text/html; charset=utf-8',
  '.css': 'text/css; charset=utf-8',
};

const args = cli(process.argv.slice(2));
const { pages, title } = build(args.repo, args.ref, args.out);
const root = resolve(args.out);

createServer((req, res) => {
  if (req.method !== 'GET' && req.method !== 'HEAD') {
    res.writeHead(405, { allow: 'GET, HEAD' }).end();
    return;
  }
  const path = decodeURIComponent(new URL(req.url ?? '/', 'http://localhost').pathname);
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
  res.writeHead(200, { 'content-type': types[extname(file)] ?? 'application/octet-stream' });
  if (req.method === 'HEAD') res.end();
  else createReadStream(file).pipe(res);
}).listen(args.port, '127.0.0.1', () => {
  console.log(`${title}: ${pages} pages, serving http://localhost:${args.port}/`);
});
