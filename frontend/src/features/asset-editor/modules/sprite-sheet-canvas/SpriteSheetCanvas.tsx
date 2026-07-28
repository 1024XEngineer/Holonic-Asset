import type { SpriteSheetCanvasProps } from "./SpriteSheetCanvas.interface";
import { getGridBounds } from "@/lib/grid-bounds";

export function SpriteSheetCanvas({ model, onEvent }: SpriteSheetCanvasProps) {
  const highlightedCells = new Set(model.selectedCellIndexes);
  const hasSelection = highlightedCells.size > 0;

  return (
    <main className="min-h-0 min-w-0 flex-1 overflow-hidden bg-[#eeece7] p-6 lg:p-8">
      <div className="flex h-full min-h-[36rem] flex-col gap-4">
        <section
          aria-label="Tileset canvas"
          className="flex min-h-0 flex-1 items-center justify-center [container-type:size]"
        >
          <div
            className="relative grid size-[min(100cqw,100cqh)] overflow-hidden bg-white"
            style={{
              gridTemplateColumns: `repeat(${model.gridSize}, minmax(0, 1fr))`,
              gridTemplateRows: `repeat(${model.gridSize}, minmax(0, 1fr))`,
            }}
          >
            {model.items.flatMap((item) => {
              if (!item.imageUrl) return [];
              const bounds = getGridBounds(item.tiles);

              return (
                <img
                  key={item.id}
                  src={item.imageUrl}
                  alt={item.label}
                  draggable={false}
                  className="pointer-events-none z-10 size-full object-fill [image-rendering:pixelated]"
                  style={{
                    gridColumn: `${bounds.x + 1} / span ${bounds.width}`,
                    gridRow: `${bounds.y + 1} / span ${bounds.height}`,
                  }}
                />
              );
            })}
            {Array.from(
              { length: model.gridSize * model.gridSize },
              (_, index) => (
                <button
                  key={index}
                  type="button"
                  aria-label={`Tile ${index + 1}`}
                  aria-pressed={highlightedCells.has(index)}
                  onClick={() =>
                    onEvent({
                      type: "cell.selection.toggled",
                      cellIndex: index,
                    })
                  }
                  style={{
                    gridColumn: (index % model.gridSize) + 1,
                    gridRow: Math.floor(index / model.gridSize) + 1,
                  }}
                  className={`relative z-20 aspect-square border-0 transition-colors ${highlightedCells.has(index) ? "bg-[#b86b70]/25" : hasSelection ? "opacity-35 hover:bg-black/5 hover:opacity-70" : "hover:bg-black/5"}`}
                />
              ),
            )}
            <div
              aria-hidden="true"
              className="pointer-events-none absolute inset-0 z-30"
            >
              {Array.from({ length: model.gridSize + 1 }, (_, index) => (
                <span
                  key={`vertical-${index}`}
                  className="absolute top-0 bottom-0 w-px bg-[#5dabb0]/80"
                  style={{ left: `${(index / model.gridSize) * 100}%` }}
                />
              ))}
              {Array.from({ length: model.gridSize + 1 }, (_, index) => (
                <span
                  key={`horizontal-${index}`}
                  className="absolute right-0 left-0 h-px bg-[#5dabb0]/80"
                  style={{ top: `${(index / model.gridSize) * 100}%` }}
                />
              ))}
            </div>
          </div>
        </section>
      </div>
    </main>
  );
}
