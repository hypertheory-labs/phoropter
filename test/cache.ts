// Imported first by every test that builds a site, so the index store's
// local cache is a temporary directory, never the real one in ~/.cache.

import { mkdtempSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

process.env.XDG_CACHE_HOME = mkdtempSync(join(tmpdir(), 'phoropter-cache-'));
