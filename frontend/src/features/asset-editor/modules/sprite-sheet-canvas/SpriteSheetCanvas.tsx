import type { SpriteSheetCanvasProps } from "./SpriteSheetCanvas.interface";

export function SpriteSheetCanvas({ model, onEvent }: SpriteSheetCanvasProps) {
  const highlightedCells = new Set(model.selectedCellIndexes);
  const hasSelection = highlightedCells.size > 0;

  return (
    <main className="min-h-0 min-w-0 flex-1 overflow-hidden bg-[#eeece7] p-6 lg:p-8">
      <div className="flex h-full min-h-[36rem] flex-col gap-4">
        <section
          aria-label="Tileset canvas"
          className="flex min-h-0 flex-1 flex-col"
        >
          <div
            className="grid min-h-0 flex-1 gap-1"
            style={{
              gridTemplateColumns: `repeat(${model.gridSize}, minmax(0, 1fr))`,
              gridTemplateRows: `repeat(${model.gridSize}, minmax(0, 1fr))`,
            }}
          >
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
                  className={`relative overflow-hidden rounded-sm border bg-[#b5ce9c] transition-all duration-200 ${highlightedCells.has(index) ? "z-10 border-[#b86b70] ring-2 ring-inset ring-[#b86b70]/60" : hasSelection ? "border-black/5 opacity-35 hover:opacity-70" : "border-black/10 hover:border-[#b86b70]/60"}`}
                >
                  <span
                    className={`absolute inset-[18%] rounded ${index % 4 === 0 ? "bg-[#6f9b6b]" : "bg-[#d9b078]"}`}
                  />
                </button>
              ),
            )}
          </div>
        </section>
      </div>
    </main>
  );
}
