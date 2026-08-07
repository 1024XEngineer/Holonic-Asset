import type { ReactNode } from "react";

import { cn } from "@/lib/utils";

export type TileSelectionGridProps = {
  gridSize: number;
  selectedCellIndexes: readonly number[];
  onToggleCell: (cellIndex: number) => void;
  ariaLabel: string;
  className?: string;
  cellClassName?: string;
  selectedCellClassName?: string;
  unselectedCellClassName?: string;
  getCellAriaLabel?: (cellIndex: number, column: number, row: number) => string;
  beforeCells?: ReactNode;
  afterCells?: ReactNode;
};

export function TileSelectionGrid({
  gridSize,
  selectedCellIndexes,
  onToggleCell,
  ariaLabel,
  className,
  cellClassName,
  selectedCellClassName = "bg-primary/35",
  unselectedCellClassName = "hover:bg-muted",
  getCellAriaLabel = (cellIndex) => `Tile ${cellIndex + 1}`,
  beforeCells,
  afterCells,
}: TileSelectionGridProps) {
  const selectedCells = new Set(selectedCellIndexes);

  if (!Number.isInteger(gridSize) || gridSize <= 0) return null;

  return (
    <div
      className={cn("relative grid overflow-hidden", className)}
      role="group"
      aria-label={ariaLabel}
      style={{
        gridTemplateColumns: `repeat(${gridSize}, minmax(0, 1fr))`,
        gridTemplateRows: `repeat(${gridSize}, minmax(0, 1fr))`,
      }}
    >
      {beforeCells}
      {Array.from({ length: gridSize * gridSize }, (_, index) => {
        const column = index % gridSize;
        const row = Math.floor(index / gridSize);
        const selected = selectedCells.has(index);

        return (
          <button
            key={index}
            type="button"
            aria-label={getCellAriaLabel(index, column, row)}
            aria-pressed={selected}
            onClick={() => onToggleCell(index)}
            style={{
              gridColumn: column + 1,
              gridRow: row + 1,
            }}
            className={cn(
              "relative z-20 aspect-square border-r border-b focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
              cellClassName,
              selected ? selectedCellClassName : unselectedCellClassName,
            )}
          />
        );
      })}
      {afterCells}
    </div>
  );
}
