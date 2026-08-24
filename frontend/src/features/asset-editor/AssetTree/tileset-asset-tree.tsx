import { ChevronDown, PackageOpen } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";

import { TileSelectionGrid } from "@/components/tile-selection-grid";
import { getGridBounds } from "@/lib/grid-bounds";
import type { TilesetItem } from "@/model";

import { AssetTree } from "./asset-tree";

export function TilesetAssetTree({
  items,
  selectedItemIds,
  isTileSelected,
  onToggleItem,
  onToggleTile,
}: {
  items: readonly TilesetItem[];
  selectedItemIds: readonly string[];
  isTileSelected: (itemId: string, tileIndex: number) => boolean;
  onToggleItem: (itemId: string) => void;
  onToggleTile: (itemId: string, tileIndex: number) => void;
}) {
  const { t } = useTranslation("editor");
  const selectedItems = new Set(selectedItemIds);

  return (
    <AssetTree
      title={t("assetTree")}
      description={t("tilesetAssetTreeDescription")}
      count={items.length}
      emptyMessage={items.length === 0 ? t("noTilesetItems") : undefined}
      contentClassName="space-y-1"
    >
      {items.map((item) => (
        <TilesetItemNode
          key={item.id}
          item={item}
          selected={selectedItems.has(item.id)}
          isTileSelected={isTileSelected}
          onToggleItem={onToggleItem}
          onToggleTile={onToggleTile}
        />
      ))}
    </AssetTree>
  );
}

function TilesetItemNode({
  item,
  selected,
  isTileSelected,
  onToggleItem,
  onToggleTile,
}: {
  item: TilesetItem;
  selected: boolean;
  isTileSelected: (itemId: string, tileIndex: number) => boolean;
  onToggleItem: (itemId: string) => void;
  onToggleTile: (itemId: string, tileIndex: number) => void;
}) {
  const { t } = useTranslation("editor");
  const [open, setOpen] = useState(false);
  const hasSelectedTile = item.tiles.some((_, tileIndex) =>
    isTileSelected(item.id, tileIndex),
  );

  return (
    <div>
      <div
        className={`flex items-center rounded-lg transition-colors ${selected ? "bg-muted text-foreground" : hasSelectedTile ? "bg-muted/50 text-foreground" : "text-muted-foreground hover:bg-muted/40 hover:text-foreground"}`}
      >
        <button
          type="button"
          aria-pressed={selected}
          onClick={() => onToggleItem(item.id)}
          className="flex min-w-0 flex-1 items-center gap-2 px-2 py-2 text-left"
        >
          <PackageOpen className="size-4 shrink-0 text-primary" />
          <span className="min-w-0 flex-1 truncate text-xs font-medium">
            {item.label}
          </span>
          <span className="font-mono text-[10px] text-muted-foreground">
            {item.tiles.length}
          </span>
        </button>
        <button
          type="button"
          aria-label={`${open ? t("collapse") : t("expand")} ${item.label}`}
          aria-expanded={open}
          onClick={() => setOpen((current) => !current)}
          className="mr-1 rounded-md p-1.5 text-muted-foreground hover:bg-muted"
        >
          <ChevronDown
            className={`size-3.5 transition-transform ${open ? "rotate-0" : "-rotate-90"}`}
          />
        </button>
      </div>
      {open ? (
        <TilesetItemGrid
          item={item}
          isTileSelected={isTileSelected}
          onToggleTile={onToggleTile}
        />
      ) : null}
    </div>
  );
}

function TilesetItemGrid({
  item,
  isTileSelected,
  onToggleTile,
}: {
  item: TilesetItem;
  isTileSelected: (itemId: string, tileIndex: number) => boolean;
  onToggleTile: (itemId: string, tileIndex: number) => void;
}) {
  const { t } = useTranslation("editor");
  if (item.tiles.length === 0) return null;

  const bounds = getGridBounds(item.tiles);
  const cellIndexes = item.tiles.map(
    ([column, row]) => (row - bounds.y) * bounds.width + (column - bounds.x),
  );
  const selectedCellIndexes = cellIndexes.filter((_, tileIndex) =>
    isTileSelected(item.id, tileIndex),
  );
  const disabledCellIndexes = Array.from(
    { length: bounds.width * bounds.height },
    (_, cellIndex) => cellIndex,
  ).filter((cellIndex) => !cellIndexes.includes(cellIndex));

  return (
    <div className="ml-8 mt-2 pb-2">
      <div
        style={{
          width: `${bounds.width * 2.25}rem`,
          height: `${bounds.height * 2.25}rem`,
        }}
      >
        <TileSelectionGrid
          gridSize={bounds.width}
          rowCount={bounds.height}
          selectedCellIndexes={selectedCellIndexes}
          disabledCellIndexes={disabledCellIndexes}
          onToggleCell={(cellIndex) => {
            const tileIndex = cellIndexes.indexOf(cellIndex);
            if (tileIndex >= 0) onToggleTile(item.id, tileIndex);
          }}
          ariaLabel={t("itemTiles", { value: item.label })}
          getCellAriaLabel={(_, column, row) =>
            t("tilesetTile", {
              value: item.label,
              column: column + 1,
              row: row + 1,
            })
          }
          className="size-full"
          selectedCellClassName="border border-border bg-muted"
          unselectedCellClassName="border border-border bg-background hover:bg-muted/40"
          disabledCellClassName="pointer-events-none border-transparent bg-transparent opacity-0"
        />
      </div>
    </div>
  );
}
