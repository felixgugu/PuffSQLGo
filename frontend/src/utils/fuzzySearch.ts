export type QuickFinderObjectType =
  | 'table'
  | 'view'
  | 'procedure'
  | 'function'
  | 'column';

export interface QuickFinderItem {
  id: string;
  schema: string;
  name: string;
  type: QuickFinderObjectType;
  database: string;
  connId: string;
  /** For `type: 'column'`: the table or view that owns the column. */
  table?: string;
}

export interface FuzzyMatchResult {
  matched: boolean;
  score: number;
  indices: number[];
}

export interface HighlightChunk {
  text: string;
  highlight: boolean;
}

export interface ScoredQuickFinderItem {
  item: QuickFinderItem;
  score: number;
  nameIndices: number[];
  schemaIndices: number[];
  /** For column items: highlight indices inside the owning table name. */
  ownerIndices: number[];
}

/**
 * Checks if character at index is a word boundary in target:
 * - First character
 * - Preceded by non-alphanumeric (e.g. '.', '_', ' ', '-')
 * - Uppercase character preceded by lowercase (CamelCase boundary)
 */
function isWordBoundary(target: string, index: number): boolean {
  if (index === 0) return true;
  const prev = target[index - 1];
  const curr = target[index];
  if (!prev || !curr) return false;

  if (/[\._\s\-]/.test(prev)) return true;
  if (/[a-z]/.test(prev) && /[A-Z]/.test(curr)) return true;
  return false;
}

/**
 * Robust fuzzy matching algorithm supporting substring, word boundary, and subsequence matching.
 */
export function fuzzyMatch(query: string, target: string): FuzzyMatchResult {
  const cleanQuery = query.trim().toLowerCase();
  const lowerTarget = target.toLowerCase();

  if (!cleanQuery) {
    return { matched: true, score: 0, indices: [] };
  }

  // 1. Exact match
  if (lowerTarget === cleanQuery) {
    return {
      matched: true,
      score: 1000,
      indices: Array.from({ length: target.length }, (_, i) => i),
    };
  }

  // 2. Exact Substring match (Very high score)
  const subIdx = lowerTarget.indexOf(cleanQuery);
  if (subIdx !== -1) {
    const indices = Array.from({ length: cleanQuery.length }, (_, i) => subIdx + i);
    let score = 500 - subIdx * 5;
    if (subIdx === 0) {
      score += 200; // Prefix bonus
    } else if (isWordBoundary(target, subIdx)) {
      score += 150; // Boundary bonus
    }
    return { matched: true, score, indices };
  }

  // 3. Subsequence fuzzy match
  let qIdx = 0;
  let tIdx = 0;
  const matchedIndices: number[] = [];
  let score = 100;
  let consecutiveCount = 0;

  while (qIdx < cleanQuery.length && tIdx < target.length) {
    if (cleanQuery[qIdx] === lowerTarget[tIdx]) {
      matchedIndices.push(tIdx);

      // Consecutive match bonus
      if (consecutiveCount > 0) {
        score += consecutiveCount * 12;
      }
      consecutiveCount++;

      // Word boundary bonus
      if (isWordBoundary(target, tIdx)) {
        score += 20;
      }

      qIdx++;
    } else {
      consecutiveCount = 0;
    }
    tIdx++;
  }

  if (qIdx === cleanQuery.length) {
    // Length penalty: penalize sprawling targets slightly
    const lengthDiff = target.length - cleanQuery.length;
    score -= Math.min(lengthDiff * 2, 80);

    return { matched: true, score: Math.max(score, 1), indices: matchedIndices };
  }

  return { matched: false, score: 0, indices: [] };
}

/**
 * Splits target string into an array of chunks with highlight flags.
 */
export function highlightMatchedChunks(target: string, indices: number[]): HighlightChunk[] {
  if (!indices || indices.length === 0) {
    return [{ text: target, highlight: false }];
  }

  const indexSet = new Set(indices);
  const chunks: HighlightChunk[] = [];
  let currentText = '';
  let currentHighlight = false;

  for (let i = 0; i < target.length; i++) {
    const isHighlight = indexSet.has(i);
    const char = target[i] ?? '';

    if (i === 0) {
      currentText = char;
      currentHighlight = isHighlight;
    } else if (isHighlight === currentHighlight) {
      currentText += char;
    } else {
      chunks.push({ text: currentText, highlight: currentHighlight });
      currentText = char;
      currentHighlight = isHighlight;
    }
  }

  if (currentText.length > 0) {
    chunks.push({ text: currentText, highlight: currentHighlight });
  }

  return chunks;
}

export interface ParsedSearchQuery {
  rawQuery: string;
  cleanQuery: string;
  explicitTypeFilter?: QuickFinderObjectType;
}

/**
 * Parses search query string for type prefix shortcuts:
 * - 't: users' or 'table: users' -> type: table
 * - 'v: sales' or 'view: sales' -> type: view
 * - 'p: proc' or 'sp: proc' or 'proc: proc' -> type: procedure
 * - 'f: fn' or 'func: fn' or 'function: fn' -> type: function
 * - 'c: id' or 'col: id' or 'column: id' -> type: column
 */
export function parseSearchQuery(input: string): ParsedSearchQuery {
  const trimmed = input.trim();
  const prefixMatch = trimmed.match(
    /^(t|table|v|view|p|sp|proc|procedure|f|fn|func|function|c|col|column):\s*(.*)$/i
  );

  if (!prefixMatch) {
    return { rawQuery: input, cleanQuery: trimmed };
  }

  const prefix = (prefixMatch[1] || '').toLowerCase();
  const cleanQuery = (prefixMatch[2] || '').trim();

  let explicitTypeFilter: QuickFinderObjectType | undefined;
  if (prefix === 't' || prefix === 'table') {
    explicitTypeFilter = 'table';
  } else if (prefix === 'v' || prefix === 'view') {
    explicitTypeFilter = 'view';
  } else if (prefix === 'p' || prefix === 'sp' || prefix === 'proc' || prefix === 'procedure') {
    explicitTypeFilter = 'procedure';
  } else if (prefix === 'f' || prefix === 'fn' || prefix === 'func' || prefix === 'function') {
    explicitTypeFilter = 'function';
  } else if (prefix === 'c' || prefix === 'col' || prefix === 'column') {
    explicitTypeFilter = 'column';
  }

  return {
    rawQuery: input,
    cleanQuery,
    explicitTypeFilter,
  };
}

/**
 * Filters and ranks database objects based on search query and type filter.
 */
export function filterAndRankQuickObjects(
  items: QuickFinderItem[],
  query: string,
  selectedType: 'all' | QuickFinderObjectType = 'all',
  maxResults = 100
): ScoredQuickFinderItem[] {
  const { cleanQuery, explicitTypeFilter } = parseSearchQuery(query);
  const effectiveType = explicitTypeFilter ?? (selectedType === 'all' ? undefined : selectedType);

  const filtered = effectiveType
    ? items.filter((item) => item.type === effectiveType)
    : items;

  if (!cleanQuery) {
    // Columns are only listed when the user explicitly asks for them: a database can hold tens of
    // thousands, and the unfiltered "all" view should stay a list of objects.
    const initial = effectiveType
      ? filtered
      : filtered.filter((item) => item.type !== 'column');
    return initial.slice(0, maxResults).map((item) => ({
      item,
      score: 0,
      nameIndices: [],
      schemaIndices: [],
      ownerIndices: [],
    }));
  }

  const results: ScoredQuickFinderItem[] = [];

  for (const item of filtered) {
    // 1. Try matching the object (or column) name directly
    const nameMatch = fuzzyMatch(cleanQuery, item.name);

    if (nameMatch.matched) {
      results.push({
        item,
        // Columns score a little lower so an object with the same name stays on top.
        score: nameMatch.score + (item.type === 'column' ? 40 : 50),
        nameIndices: nameMatch.indices,
        schemaIndices: [],
        ownerIndices: [],
      });
      continue;
    }

    // 2. Columns also match through their owning table: "Orders.OrderId" / "OrderId".
    if (item.type === 'column' && item.table) {
      const ownerTarget = `${item.table}.${item.name}`;
      const ownerMatch = fuzzyMatch(cleanQuery, ownerTarget);

      if (ownerMatch.matched) {
        const ownerLen = item.table.length;
        const ownerIndices: number[] = [];
        const nameIndices: number[] = [];

        for (const idx of ownerMatch.indices) {
          if (idx < ownerLen) {
            ownerIndices.push(idx);
          } else if (idx > ownerLen) {
            nameIndices.push(idx - ownerLen - 1);
          }
        }

        results.push({
          item,
          score: ownerMatch.score,
          nameIndices,
          schemaIndices: [],
          ownerIndices,
        });
        continue;
      }

      // 3. Last chance: the fully qualified schema.table.column (no per-segment highlight).
      const qualified = `${item.schema}.${item.table}.${item.name}`;
      const qualifiedMatch = fuzzyMatch(cleanQuery, qualified);
      if (qualifiedMatch.matched) {
        results.push({
          item,
          score: qualifiedMatch.score,
          nameIndices: [],
          schemaIndices: [],
          ownerIndices: [],
        });
      }
      continue;
    }

    // 4. Try matching full qualified schema.name
    const fullTarget = `${item.schema}.${item.name}`;
    const fullMatch = fuzzyMatch(cleanQuery, fullTarget);

    if (fullMatch.matched) {
      const schemaLen = item.schema.length;
      const schemaIndices: number[] = [];
      const nameIndices: number[] = [];

      for (const idx of fullMatch.indices) {
        if (idx < schemaLen) {
          schemaIndices.push(idx);
        } else if (idx > schemaLen) {
          nameIndices.push(idx - schemaLen - 1);
        }
      }

      results.push({
        item,
        score: fullMatch.score,
        nameIndices,
        schemaIndices,
        ownerIndices: [],
      });
    }
  }

  // Rank by score descending, then by type priority, then by name length ascending
  const typeWeight: Record<QuickFinderObjectType, number> = {
    table: 4,
    view: 3,
    procedure: 2,
    function: 1,
    column: 0,
  };

  results.sort((a, b) => {
    if (b.score !== a.score) {
      return b.score - a.score;
    }
    const weightDiff = (typeWeight[b.item.type] || 0) - (typeWeight[a.item.type] || 0);
    if (weightDiff !== 0) {
      return weightDiff;
    }
    if (a.item.name.length !== b.item.name.length) {
      return a.item.name.length - b.item.name.length;
    }
    return a.item.name.localeCompare(b.item.name);
  });

  return results.slice(0, maxResults);
}
