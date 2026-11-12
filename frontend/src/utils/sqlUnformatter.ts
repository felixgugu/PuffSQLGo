import { parseSqlLines } from './sqlStatementExtractor';

/**
 * Compacts a SQL fragment into a single line:
 *
 * - Whitespace outside literals/comments collapses to one space.
 * - String literals, bracket identifiers and double-quoted identifiers keep every byte, including
 *   their internal spaces and newlines.
 * - Block comments keep their text but lose their internal formatting whitespace.
 * - Line comments keep their terminating newline; removing it would comment out the SQL after them.
 * - `GO` batch separators stay on their own line because SQL Server treats them as line directives.
 *
 * Keywords are never case-folded; this is a whitespace transform, not a formatter.
 */
export function unformatSql(sql: string): string {
  if (!sql || !sql.trim()) return sql;

  const { lines, lineMetas } = parseSqlLines(sql);
  const goLines = new Set(
    lineMetas.filter((meta) => meta.isGo).map((meta) => meta.lineIndex)
  );

  const output: string[] = [];
  let chunk: string[] = [];

  const flushChunk = () => {
    const collapsed = collapseSqlChunk(chunk.join('\n'));
    if (collapsed) output.push(collapsed);
    chunk = [];
  };

  for (let index = 0; index < lines.length; index++) {
    if (goLines.has(index)) {
      flushChunk();
      output.push((lines[index] ?? '').trim());
      continue;
    }
    chunk.push(lines[index] ?? '');
  }
  flushChunk();

  return output.join('\n').trim();
}

/** Collapses one GO-delimited batch; the scanner never touches literal/comment interiors. */
function collapseSqlChunk(text: string): string {
  let output = '';
  let pendingSpace = false;
  let i = 0;
  const length = text.length;

  const pushPendingSpace = () => {
    if (!pendingSpace) return;
    pendingSpace = false;
    if (output.length === 0) return;
    if (output.endsWith(' ') || output.endsWith('\n')) return;
    output += ' ';
  };

  while (i < length) {
    const ch = text[i]!;
    const next = i + 1 < length ? text[i + 1]! : '';

    // Line comment: preserve the text and its terminating newline.
    if (ch === '-' && next === '-') {
      pushPendingSpace();
      let end = i;
      while (end < length && text[end] !== '\n' && text[end] !== '\r') end++;
      output += text.slice(i, end);
      if (end < length) {
        if (text[end] === '\r' && text[end + 1] === '\n') end++;
        output += '\n';
        i = end + 1;
        pendingSpace = false;
      } else {
        i = end;
      }
      continue;
    }

    // Block comment: keep the comment, collapse its internal formatting whitespace.
    if (ch === '/' && next === '*') {
      pushPendingSpace();
      const comment = readBlockComment(text, i);
      output += comment.text;
      i = comment.nextIndex;
      pendingSpace = false;
      continue;
    }

    // Single-quoted string (including N'...'): preserve every byte.
    if (ch === "'") {
      pushPendingSpace();
      const literal = readSingleQuoted(text, i);
      output += literal.text;
      i = literal.nextIndex;
      pendingSpace = false;
      continue;
    }

    // Bracket identifier: preserve every byte.
    if (ch === '[') {
      pushPendingSpace();
      const identifier = readBracketQuoted(text, i);
      output += identifier.text;
      i = identifier.nextIndex;
      pendingSpace = false;
      continue;
    }

    // Double-quoted identifier: preserve every byte.
    if (ch === '"') {
      pushPendingSpace();
      const identifier = readDoubleQuoted(text, i);
      output += identifier.text;
      i = identifier.nextIndex;
      pendingSpace = false;
      continue;
    }

    if (isSqlWhitespace(ch)) {
      pendingSpace = true;
      i++;
      continue;
    }

    pushPendingSpace();
    output += ch;
    i++;
  }

  return output.trim();
}

function isSqlWhitespace(ch: string): boolean {
  return ch === ' ' || ch === '\t' || ch === '\n' || ch === '\r' || ch === '\f' || ch === '\v';
}

interface ScannedText {
  text: string;
  nextIndex: number;
}

function readSingleQuoted(text: string, start: number): ScannedText {
  let i = start + 1;
  while (i < text.length) {
    if (text[i] === "'") {
      if (text[i + 1] === "'") {
        i += 2;
        continue;
      }
      i++;
      break;
    }
    i++;
  }
  return { text: text.slice(start, i), nextIndex: i };
}

function readBracketQuoted(text: string, start: number): ScannedText {
  let i = start + 1;
  while (i < text.length) {
    if (text[i] === ']') {
      if (text[i + 1] === ']') {
        i += 2;
        continue;
      }
      i++;
      break;
    }
    i++;
  }
  return { text: text.slice(start, i), nextIndex: i };
}

function readDoubleQuoted(text: string, start: number): ScannedText {
  let i = start + 1;
  while (i < text.length) {
    if (text[i] === '"') {
      if (text[i + 1] === '"') {
        i += 2;
        continue;
      }
      i++;
      break;
    }
    i++;
  }
  return { text: text.slice(start, i), nextIndex: i };
}

/**
 * Block comments may nest in SQL Server. The content is preserved except for runs of whitespace,
 * which become one space so the resulting statement can stay on a single line.
 */
function readBlockComment(text: string, start: number): ScannedText {
  let i = start + 2;
  let depth = 1;
  let content = '/*';
  let pendingSpace = false;

  const pushPendingSpace = () => {
    if (!pendingSpace) return;
    pendingSpace = false;
    if (content.endsWith(' ')) return;
    content += ' ';
  };

  while (i < text.length) {
    const ch = text[i]!;
    const next = i + 1 < text.length ? text[i + 1]! : '';

    if (ch === '/' && next === '*') {
      pushPendingSpace();
      content += '/*';
      depth++;
      i += 2;
      continue;
    }

    if (ch === '*' && next === '/') {
      depth--;
      if (depth === 0) {
        content += '*/';
        i += 2;
        break;
      }
      pushPendingSpace();
      content += '*/';
      i += 2;
      continue;
    }

    if (isSqlWhitespace(ch)) {
      pendingSpace = true;
      i++;
      continue;
    }

    pushPendingSpace();
    content += ch;
    i++;
  }

  return { text: content, nextIndex: i };
}
