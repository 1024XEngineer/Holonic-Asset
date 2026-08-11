import { useMemo } from "react";

import { TileSelectionGrid } from "@/components/tile-selection-grid";
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
      <div className="size-32 justify-self-center border border-primary/70 bg-background">
        <TileSelectionGrid
          gridSize={gridSize}
          selectedCellIndexes={shape.map(([x, y]) => y * gridSize + x)}
          onToggleCell={(index) =>
            toggleTile(index % gridSize, Math.floor(index / gridSize))
          }
          ariaLabel="Item shape"
          className="size-full"
          cellClassName="border-primary/70"
          getCellAriaLabel={(_, x, y) => `Tile column ${x + 1}, row ${y + 1}`}
        />
      </div>
      <p className="text-center text-xs leading-4 text-muted-foreground">
        Click tiles to define the item footprint.
      </p>
    </div>
  );
}
