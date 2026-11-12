import type { Tabulator, TabulatorCellComponent } from 'tabulator-tables';

/** Minimal shape of the range module internals this project relies on. */
interface RangeModuleInternals {
  setDefaultRange?: () => void;
  layoutElement?: (visibleRows?: boolean) => void;
}

export const CLEAR_GRID_SELECTION_EVENT = 'sqlight:clear-grid-selection';

/** Internal cell returned by the range module; it knows how to expose its public component. */
interface InternalRangeCell {
  getComponent?: () => TabulatorCellComponent;
}

/**
 * The cell currently owning the range focus. Tabulator keeps this inside the range module; it is
 * the same cell its own Enter-to-edit shortcut edits, so F2 can mirror that behaviour.
 */
export function activeRangeCell(
  table: Tabulator | null | undefined
): TabulatorCellComponent | null {
  if (!table) return null;

  const selectRange = (
    table as unknown as {
      modules?: { selectRange?: { getActiveCell?: () => InternalRangeCell | null } };
    }
  ).modules?.selectRange;

  return selectRange?.getActiveCell?.()?.getComponent?.() ?? null;
}

/**
 * Opens the inline editor for a cell component. Tabulator 6 keeps `editCell` on the internal edit
 * module; the public CellComponent intentionally has no `edit()` method, so calling `cell.edit()`
 * (or passing the public component) does nothing.
 */
export function editCell(
  table: Tabulator | null | undefined,
  cell: TabulatorCellComponent | null | undefined
): boolean {
  if (!table || !cell) return false;

  const editModule = (
    table as unknown as {
      modules?: { edit?: { editCell?: (cell: unknown) => void } };
    }
  ).modules?.edit;
  if (!editModule?.editCell) return false;

  const internalCell =
    (cell as unknown as { _getSelf?: () => unknown })._getSelf?.() ?? cell;

  editModule.editCell(internalCell);
  return true;
}

/** Dispatches a global event to instruct active data grids to clear their range selection. */
export function dispatchClearGridSelection(): void {
  if (typeof window !== 'undefined') {
    window.dispatchEvent(new CustomEvent(CLEAR_GRID_SELECTION_EVENT));
  }
}


/**
 * Drops every active range.
 *
 * Tabulator exposes `addRange()` but no public "clear ranges" call. Removing the last range with
 * `RangeComponent.remove()` makes the range module auto-create an empty placeholder range whose
 * bounds are `NaN`; a later layout pass then throws while trying to resolve its cells. The module's
 * own `setDefaultRange()` (used internally on `data-processed`) tears the ranges down without that
 * placeholder, and with `selectableRangeInitializeDefault: false` it leaves none behind.
 */
export function clearGridRanges(table: Tabulator | null | undefined): void {
  if (!table) return;

  const selectRange = (
    table as unknown as { modules?: { selectRange?: RangeModuleInternals } }
  ).modules?.selectRange;

  if (selectRange?.setDefaultRange) {
    try {
      selectRange.setDefaultRange();
      // Re-paint the rendered cells so stale `.tabulator-range-selected` classes disappear.
      selectRange.layoutElement?.(true);
      return;
    } catch {
      // Fall through to the public API if the internals ever change shape.
    }
  }

  try {
    for (const range of table.getRanges()) {
      range.remove();
    }
  } catch {
    // A grid that is already torn down has nothing left to clear.
  }
}
