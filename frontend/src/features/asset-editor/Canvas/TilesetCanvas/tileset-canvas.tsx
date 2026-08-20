import { Grid3X3 } from "lucide-react";
import { useTranslation } from "react-i18next";

import { TileSelectionGrid } from "@/components/tile-selection-grid";
import { getGridBounds } from "@/lib/grid-bounds";
import type { TilesetItem } from "@/model";

import type { TilesetCanvasProps } from "./TilesetCanvas.interface";
import {
  getTilesetCellIndex,
  isValidGridSize,
} from "./TilesetCanvasStateMachine";

export function TilesetCanvas({ model, onEvent }: TilesetCanvasProps) {
  const { t } = useTranslation("editor");
  const gridSize = isValidGridSize(model.gridSize) ? model.gridSize : 0;
  const highlightedCells = new Set(
    model.selectedCellIndexes.filter(
      (cellIndex) =>
        Number.isInteger(cellIndex) &&
        cellIndex >= 0 &&
        cellIndex < gridSize * gridSize,
    ),
  );
  const hasSelection = highlightedCells.size > 0;

  return (
    <main className="min-h-0 min-w-0 flex-1 overflow-hidden bg-[#eeece7] p-4 sm:p-6 lg:p-8">
      <section
        aria-label={t("tilesetCanvas")}
        className="flex h-full min-h-[24rem] items-center justify-center [container-type:size] lg:min-h-[36rem]"
      >
        {gridSize === 0 ? (
          <EmptyTilesetCanvas />
        ) : (
          <TileSelectionGrid
            gridSize={gridSize}
            selectedCellIndexes={[...highlightedCells]}
            onToggleCell={(gridCellIndex) =>
              onEvent({ type: "cell.selection.toggled", gridCellIndex })
            }
            ariaLabel={t("tilesetGrid")}
            className="size-[min(100cqw,100cqh)] border border-[#5dabb0] bg-white shadow-[0_18px_50px_rgb(45_41_35/0.14)]"
            cellClassName="border-0 transition-[background-color,opacity] focus-visible:outline-2 focus-visible:outline-offset-[-2px] focus-visible:outline-[#b86b70]"
            selectedCellClassName="bg-[#b86b70]/30"
            unselectedCellClassName={
              hasSelection ? "opacity-40 hover:bg-black/5" : "hover:bg-black/5"
            }
            beforeCells={
              <>
                {model.items.flatMap((item) => {
                  if (item.tileUrls) {
                    return item.tiles.flatMap((tile, tileIndex) => {
                      const url = item.tileUrls?.[tileIndex];
                      if (!url) return [];

                      return [
                        <img
                          key={`${item.id}:${tile[0]}:${tile[1]}`}
                          src={url}
                          alt=""
                          draggable={false}
                          className="pointer-events-none z-10 size-full object-fill [image-rendering:pixelated]"
                          style={{
                            gridColumn: tile[0] + 1,
                            gridRow: tile[1] + 1,
                          }}
                        />,
                      ];
                    });
                  }

                  const bounds = getRenderableItemBounds(item, gridSize);
                  if (!item.imageUrl || !bounds) return [];

                  return (
                    <img
                      key={item.id}
                      src={item.imageUrl}
                      alt=""
                      draggable={false}
                      className="pointer-events-none z-10 size-full object-fill [image-rendering:pixelated]"
                      style={{
                        gridColumn: `${bounds.x + 1} / span ${bounds.width}`,
                        gridRow: `${bounds.y + 1} / span ${bounds.height}`,
                      }}
                    />
                  );
                })}
              </>
            }
            afterCells={
              <div
                aria-hidden="true"
                className="pointer-events-none absolute inset-0 z-30 opacity-80"
                style={{
                  backgroundImage:
                    "linear-gradient(to right, #5dabb0 1px, transparent 1px), linear-gradient(to bottom, #5dabb0 1px, transparent 1px)",
                  backgroundSize: `${100 / gridSize}% ${100 / gridSize}%`,
                }}
              />
            }
          />
        )}
      </section>
      <p className="sr-only" aria-live="polite">
        {hasSelection
          ? t("tilesSelected", { count: highlightedCells.size })
          : t("noTilesSelected")}
      </p>
    </main>
  );
}

function getRenderableItemBounds(item: TilesetItem, gridSize: number) {
  if (
    item.tiles.length === 0 ||
    item.tiles.some(
      (coordinate) => getTilesetCellIndex(coordinate, gridSize) === undefined,
    )
  ) {
    return undefined;
  }

  return getGridBounds(item.tiles);
}

function EmptyTilesetCanvas() {
  const { t } = useTranslation("editor");
  return (
    <div className="flex flex-col items-center justify-center gap-2 text-[#47656a]">
      <Grid3X3 className="size-5" aria-hidden="true" />
      <p className="text-xs font-medium">{t("noTilesetGrid")}</p>
    </div>
  );
}
