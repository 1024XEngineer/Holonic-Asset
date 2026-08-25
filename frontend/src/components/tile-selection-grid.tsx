import type { ReactNode } from "react";

import { cn } from "@/lib/utils";

export type TileSelectionGridProps = {
  gridSize: number;
  rowCount?: number;
  selectedCellIndexes: readonly number[];
  disabledCellIndexes?: readonly number[];
  onToggleCell?: (cellIndex: number) => void;
  ariaLabel: string;
  className?: string;
  cellClassName?: string;
  selectedCellClassName?: string;
  unselectedCellClassName?: string;
  disabledCellClassName?: string;
  getCellAriaLabel?: (cellIndex: number, column: number, row: number) => string;
  beforeCells?: ReactNode;
  afterCells?: ReactNode;
};

export function TileSelectionGrid({
  gridSize,
  rowCount = gridSize,
  selectedCellIndexes,
  disabledCellIndexes = [],
  onToggleCell,
  ariaLabel,
  className,
  cellClassName,
  selectedCellClassName = "bg-primary/35",
  unselectedCellClassName = "hover:bg-muted",
  disabledCellClassName = "cursor-not-allowed opacity-30",
  getCellAriaLabel = (cellIndex) => `Tile ${cellIndex + 1}`,
  beforeCells,
  afterCells,
}: TileSelectionGridProps) {
  const selectedCells = new Set(selectedCellIndexes);
  const disabledCells = new Set(disabledCellIndexes);

  if (
    !Number.isInteger(gridSize) ||
    gridSize <= 0 ||
    !Number.isInteger(rowCount) ||
    rowCount <= 0
  )
    return null;

  return (
    <div
      className={cn("relative grid overflow-hidden", className)}
      role="group"
      aria-label={ariaLabel}
      style={{
        gridTemplateColumns: `repeat(${gridSize}, minmax(0, 1fr))`,
        gridTemplateRows: `repeat(${rowCount}, minmax(0, 1fr))`,
      }}
    >
      {beforeCells}
      {Array.from({ length: gridSize * rowCount }, (_, index) => {
        const column = index % gridSize;
        const row = Math.floor(index / gridSize);
        const selected = selectedCells.has(index);
        const disabled = disabledCells.has(index);
        let cellStateClassName = unselectedCellClassName;
        if (disabled) cellStateClassName = disabledCellClassName;
        else if (selected) cellStateClassName = selectedCellClassName;

        return (
          <button
            key={index}
            type="button"
            disabled={disabled}
            aria-label={getCellAriaLabel(index, column, row)}
            aria-pressed={selected}
            onClick={() => onToggleCell?.(index)}
            style={{
              gridColumn: column + 1,
              gridRow: row + 1,
            }}
            className={cn(
              "relative z-20 aspect-square border-r border-b focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
              cellClassName,
              cellStateClassName,
            )}
          />
        );
      })}
      {afterCells}
    </div>
  );
}
