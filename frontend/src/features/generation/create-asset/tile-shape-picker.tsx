import { useMemo } from "react";

import { cn } from "@/lib/utils";
import type { TileShapeCell } from "@/model/generation";

const gridSize = 4;

export function TileShapePicker({
  cells,
  onChange,
}: {
  cells: TileShapeCell[];
  onChange: (cells: TileShapeCell[]) => void;
}) {
  const selectedCells = useMemo(
    () => new Set(cells.map(([x, y]) => `${x}:${y}`)),
    [cells],
  );

  const toggleCell = (x: number, y: number) => {
    const isSelected = selectedCells.has(`${x}:${y}`);
    const nextCells = isSelected
      ? cells.filter(([cellX, cellY]) => cellX !== x || cellY !== y)
      : [...cells, [x, y] as TileShapeCell];

    onChange(nextCells);
  };

  return (
    <div className="grid w-52 justify-self-center gap-2">
      <div className="flex items-center justify-between gap-3">
        <p className="text-xs font-semibold uppercase tracking-[0.08em] text-muted-foreground">
          Occupied tiles
        </p>
        <span className="font-mono text-xs text-muted-foreground">
          {cells.length} {cells.length === 1 ? "tile" : "tiles"}
        </span>
      </div>
      <div
        className="grid size-32 justify-self-center overflow-hidden border border-primary/70 bg-background"
        role="group"
        aria-label="Tile item shape"
        style={{ gridTemplateColumns: `repeat(${gridSize}, minmax(0, 1fr))` }}
      >
        {Array.from({ length: gridSize * gridSize }, (_, index) => {
          const x = index % gridSize;
          const y = Math.floor(index / gridSize);
          const isSelected = selectedCells.has(`${x}:${y}`);

          return (
            <button
              key={`${x}:${y}`}
              type="button"
              aria-label={`Tile column ${x + 1}, row ${y + 1}`}
              aria-pressed={isSelected}
              onClick={() => toggleCell(x, y)}
              className={cn(
                "aspect-square border-r border-b border-primary/70 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
                isSelected ? "bg-primary/35" : "hover:bg-muted",
              )}
            />
          );
        })}
      </div>
      <p className="text-center text-xs leading-4 text-muted-foreground">
        Click cells to define the item footprint.
      </p>
    </div>
  );
}
