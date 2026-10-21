/**
 * Shared source-scanning helpers for the contract tests, so a change to how the suite reads the
 * tree happens in one place. `scripts/test.mjs` only treats `tests/*.test.ts` as entry points, so
 * this module is bundled into whichever tests import it and never runs on its own.
 */
import { existsSync, readFileSync, readdirSync } from 'node:fs';
import { dirname, resolve } from 'node:path';

/** Repository root: the suite always runs from there (`npm test`). */
export const repoRoot = process.cwd();
/** Go/Wails project root, one level above the Vite frontend package. */
export const projectRoot = dirname(repoRoot);

export function projectPath(relativePath: string): string {
  return resolve(projectRoot, relativePath);
}

export function readProjectSource(relativePath: string): string {
  return readFileSync(projectPath(relativePath), 'utf-8');
}

export function readSource(relativePath: string): string {
  return readFileSync(resolve(repoRoot, relativePath), 'utf-8');
}

export interface ReadCodeOptions {
  /** Also drop `//` line comments. Off by default: a selector or note may live in one. */
  lineComments?: boolean;
}

/** Source with comments removed, so prose in a comment is never mistaken for markup. */
export function readCode(relativePath: string, options: ReadCodeOptions = {}): string {
  const source = readSource(relativePath)
    .replace(/\/\*[\s\S]*?\*\//g, '')
    .replace(/<!--[\s\S]*?-->/g, '');
  return options.lineComments ? source.replace(/(^|\s)\/\/.*$/gm, '') : source;
}

/**
 * Every frontend source file the cross-file guards scan. Scanning the filesystem keeps the suite
 * useful before the first commit and does not require a Git working tree.
 */
export function trackedSources(): string[] {
  const files: string[] = [];
  const visit = (relativeDir: string) => {
    const absolute = resolve(repoRoot, relativeDir);
    if (!existsSync(absolute)) return;
    for (const entry of readdirSync(absolute, { withFileTypes: true })) {
      const relative = `${relativeDir}/${entry.name}`;
      if (entry.isDirectory()) visit(relative);
      else if (/\.(vue|ts|js|mjs|css)$/.test(entry.name)) files.push(relative);
    }
  };
  visit('src');
  return files.sort();
}

export function countOccurrences(source: string, needle: string): number {
  return source.split(needle).length - 1;
}
