// `phoropter index --check <program>` (seersucker): runs an indexer on a
// repository at HEAD and checks what it prints against indexers/README.md,
// so whoever writes an indexer, person or agent, can tell when it's done.

import { removeDir, type Repo } from './git.ts';
import { describe, runIndexer, type IndexerOutput } from './indexers.ts';
import { isScipSymbol } from './scip.ts';

// The last name in a symbol's descriptors: `…/docScreen#annotate().` is
// `annotate`, `…/\`mock-panel.ts\`/MockPanel#` is `MockPanel`, and
// `…/OrderService#\`.ctor\`().` is `.ctor`. A backticked name is read whole,
// with a doubled backtick standing for one, as SCIP escapes them.
export function lastName(symbol: string): string | undefined {
  const d = symbol.split(' ').slice(4).join(' ');
  const names: string[] = [];
  for (let i = 0; i < d.length; ) {
    if (d[i] === '`') {
      let name = '';
      for (i++; i < d.length; i++) {
        if (d[i] === '`' && d[i + 1] === '`') name += d[i++];
        else if (d[i] === '`') break;
        else name += d[i];
      }
      names.push(name);
      i++;
    } else if (/[\w$+-]/.test(d[i])) {
      let name = '';
      while (i < d.length && /[\w$+-]/.test(d[i])) name += d[i++];
      names.push(name);
    } else i++;
  }
  return names.at(-1);
}

// The name check applies to names a source file spells as written. Anything
// else (`.ctor`, `this[]`, `<constructor>` once its brackets are off) is a
// name an indexer made up, and isn't looked for.
const IDENTIFIER = /^[A-Za-z_$][\w$]*$/;
const ALLOWED_MISSES = 0.05;

// How a name can appear in the source. Indexers name some things the source
// never spells: scip-typescript's `<constructor>` is `constructor`, and an
// anonymous object's properties get a numeric suffix (`encoding0`).
function spellings(name: string): string[] {
  const out = [name];
  const bare = name.replace(/^<(.*)>$/, '$1');
  if (bare !== name) out.push(bare);
  const undigited = bare.replace(/\d+$/, '');
  if (undigited && undigited !== bare) out.push(undigited);
  return out;
}

export function checkIndexer(program: { command: string[]; cwd?: string }, repo: Repo): { ok: boolean; report: string[] } {
  const { command, cwd } = program;
  const problems: string[] = [];
  const report: string[] = [];
  let about;
  try {
    about = describe(command, cwd);
  } catch (err) {
    return { ok: false, report: [(err as Error).message] };
  }
  for (const field of ['indexer', 'version', 'scheme'] as const) {
    if (typeof about[field] !== 'string' || !about[field]) problems.push(`--describe has no ${field}`);
  }
  if (!Array.isArray(about.extensions) || !about.extensions.length) problems.push('--describe lists no extensions, so phoropter symbols can never pick it');
  report.push(`${about.indexer} ${about.version}, scheme ${about.scheme}, for ${about.extensions.join(' ')}`);

  const sha = repo.resolve('HEAD')!;
  const tree = repo.checkout(sha);
  let out: IndexerOutput;
  try {
    out = runIndexer({ command, cwd }, tree);
  } catch (err) {
    return { ok: false, report: [...report, `running it on ${repo.short(sha)} failed: ${(err as Error).message}`] };
  } finally {
    removeDir(tree);
  }
  if (out.scheme !== about.scheme) problems.push(`it prints scheme ${out.scheme}, but --describe says ${about.scheme}`);
  if (typeof out.indexer !== 'string' || typeof out.version !== 'string') problems.push('its output has no indexer or version');
  if (!out.packages || typeof out.packages !== 'object') return { ok: false, report: [...report, 'its output has no packages object'] };

  let count = 0;
  let named = 0;
  const misses: string[] = [];
  const lines = new Map<string, string[] | undefined>();
  for (const [pkg, { definitions }] of Object.entries(out.packages)) {
    for (const [symbol, value] of Object.entries(definitions ?? {})) {
      count++;
      const say = (why: string) => problems.length < 200 && problems.push(`${symbol}: ${why}`);
      const parts = symbol.split(' ');
      if (!isScipSymbol(symbol)) say("isn't a symbol in SCIP's grammar (scheme manager package version descriptors)");
      else if (parts[0] !== about.scheme) say(`scheme ${parts[0]}, not ${about.scheme}`);
      else if (parts[2] !== pkg) say(`package ${parts[2]}, but it's listed under ${pkg}`);
      else if (parts[3] !== '.') say('its version should be written as "."');
      if (!Array.isArray(value) || value.length !== 3) {
        say('should be [file, startLine, endLine]');
        continue;
      }
      const [file, start, end] = value;
      if (!lines.has(file)) lines.set(file, repo.show(sha, file)?.split('\n'));
      const text = lines.get(file);
      if (!text) {
        say(`${file} isn't a file at ${repo.short(sha)}; paths are relative to the repository's root`);
        continue;
      }
      if (!Number.isInteger(start) || !Number.isInteger(end) || start < 1 || end < start || end > text.length) {
        say(`lines ${start}–${end} aren't a range in ${file} (${text.length} lines; 1-based, inclusive)`);
        continue;
      }
      const name = lastName(symbol);
      const range = text.slice(start - 1, end).join('\n');
      // A namespace (a package or a file, ending in `/`) is named by its path,
      // which the source doesn't spell.
      const plain = name ? spellings(name).filter((n) => IDENTIFIER.test(n)) : [];
      if (!name || symbol.endsWith('/') || !plain.length) continue;
      named++;
      if (!plain.some((n) => range.includes(n))) misses.push(`${symbol}: "${name}" isn't on lines ${start}–${end} of ${file}`);
    }
  }
  report.push(`${Object.keys(out.packages).length} package(s), ${count} definition(s) at ${repo.short(sha)}`);
  // A few names spelled differently in source (an operator's op_Addition is
  // `operator +`) are notes. Many are a range bug, which misses nearly all.
  if (misses.length > named * ALLOWED_MISSES) problems.push(...misses);
  else if (misses.length) report.push(`${misses.length} of ${named} name(s) aren't spelled in their ranges, within the ${ALLOWED_MISSES * 100}% allowed for names a language spells differently:`, ...misses.slice(0, 10).map((m) => `  ${m}`));
  if (!count) problems.push('it found no definitions at all');
  if (problems.length) report.push(`${problems.length} problem(s):`, ...problems.slice(0, 20).map((p) => `  ${p}`));
  else report.push('passes: every definition is a symbol in its scheme, in a file at HEAD, on lines that hold its name.');
  return { ok: !problems.length, report };
}
