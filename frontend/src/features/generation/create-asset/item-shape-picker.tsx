import { useMemo } from "react";

import { cn } from "@/lib/utils";
import type { ItemTile } from "@/model/item-tile";

const gridSize = 4;

export function ItemShapePicker({
  shape,
  onChange,
}: {
  shape: ItemTile[];
  onChange: (shape: ItemTile[]) => void;
}) {
  const selectedTiles = useMemo(
    () => new Set(shape.map(([x, y]) => `${x}:${y}`)),
    [shape],
  );

  const toggleTile = (x: number, y: number) => {
    const isSelected = selectedTiles.has(`${x}:${y}`);
    const nextShape = isSelected
      ? shape.filter(([tileX, tileY]) => tileX !== x || tileY !== y)
      : [...shape, [x, y] as ItemTile];

    onChange(nextShape);
  };

  return (
    <div className="grid w-52 justify-self-center gap-2">
      <div className="flex items-center justify-between gap-3">
        <p className="text-xs font-semibold uppercase tracking-[0.08em] text-muted-foreground">
          Occupied tiles
        </p>
        <span className="font-mono text-xs text-muted-foreground">
          {shape.length} {shape.length === 1 ? "tile" : "tiles"}
        </span>
      </div>
      <div
        className="grid size-32 justify-self-center overflow-hidden border border-primary/70 bg-background"
        role="group"
        aria-label="Item shape"
        style={{ gridTemplateColumns: `repeat(${gridSize}, minmax(0, 1fr))` }}
      >
        {Array.from({ length: gridSize * gridSize }, (_, index) => {
          const x = index % gridSize;
          const y = Math.floor(index / gridSize);
          const isSelected = selectedTiles.has(`${x}:${y}`);

          return (
            <button
              key={`${x}:${y}`}
              type="button"
              aria-label={`Tile column ${x + 1}, row ${y + 1}`}
              aria-pressed={isSelected}
              onClick={() => toggleTile(x, y)}
              className={cn(
                "aspect-square border-r border-b border-primary/70 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
                isSelected ? "bg-primary/35" : "hover:bg-muted",
              )}
            />
          );
        })}
      </div>
      <p className="text-center text-xs leading-4 text-muted-foreground">
        Click tiles to define the item footprint.
      </p>
    </div>
  );
}
