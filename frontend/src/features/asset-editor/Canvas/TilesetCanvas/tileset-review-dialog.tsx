import { useTranslation } from "react-i18next";

import { TileSelectionGrid } from "@/components/tile-selection-grid";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { getGridBounds } from "@/lib/grid-bounds";
import type { TilesetItem } from "@/model";

import type { TilesetItemReview } from "./TilesetCanvas.interface";

type TilesetReviewDialogProps = {
  item: TilesetItemReview;
  isResolving: boolean;
  onClose: () => void;
  onResolve: (applied: boolean) => void;
};

export function TilesetReviewDialog({
  item,
  isResolving,
  onClose,
  onResolve,
}: TilesetReviewDialogProps) {
  const { t } = useTranslation("editor");

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !isResolving) onClose();
      }}
    >
      <DialogContent className="sm:max-w-3xl" showCloseButton={!isResolving}>
        <DialogHeader>
          <DialogTitle>
            {t("tilesetReviewTitle", { value: item.candidateItem.label })}
          </DialogTitle>
          <DialogDescription>{t("tilesetReviewDescription")}</DialogDescription>
        </DialogHeader>
        <div className="grid gap-3 sm:grid-cols-2">
          <ItemComparisonPanel label={t("current")} item={item.currentItem} />
          <ItemComparisonPanel
            label={t("generated")}
            item={item.candidateItem}
          />
        </div>
        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            disabled={isResolving}
            onClick={() => onResolve(false)}
          >
            {t("cancel")}
          </Button>
          <Button
            type="button"
            disabled={isResolving}
            onClick={() => onResolve(true)}
          >
            {t("applyGeneration")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function ItemComparisonPanel({
  label,
  item,
}: {
  label: string;
  item: TilesetItem;
}) {
  const bounds = item.tiles.length > 0 ? getGridBounds(item.tiles) : null;

  return (
    <section className="rounded-lg border bg-muted/25 p-3">
      <h3 className="mb-3 text-xs font-semibold text-muted-foreground">
        {label}
      </h3>
      <div className="flex h-64 items-center justify-center overflow-hidden rounded-md bg-background p-4 shadow-inner">
        {bounds ? (
          <div
            className={
              bounds.width >= bounds.height
                ? "w-full max-w-full"
                : "h-full max-h-full"
            }
            style={{ aspectRatio: `${bounds.width} / ${bounds.height}` }}
          >
            <TileSelectionGrid
              gridSize={bounds.width}
              rowCount={bounds.height}
              selectedCellIndexes={[]}
              disabledCellIndexes={Array.from(
                { length: bounds.width * bounds.height },
                (_, index) => index,
              )}
              ariaLabel={label}
              className="size-full border border-border bg-white"
              cellClassName="border-border/70"
              disabledCellClassName="cursor-default"
              beforeCells={<ItemImages item={item} />}
            />
          </div>
        ) : null}
      </div>
    </section>
  );
}

function ItemImages({ item }: { item: TilesetItem }) {
  const bounds = getGridBounds(item.tiles);

  if (item.tileUrls) {
    return item.tiles.map(([x, y], tileIndex) => {
      const url = item.tileUrls?.[tileIndex];
      return url ? (
        <img
          key={`${x}:${y}`}
          src={url}
          alt=""
          draggable={false}
          className="pointer-events-none z-10 size-full object-fill [image-rendering:pixelated]"
          style={{
            gridColumn: x - bounds.x + 1,
            gridRow: y - bounds.y + 1,
          }}
        />
      ) : null;
    });
  }

  return item.imageUrl ? (
    <img
      src={item.imageUrl}
      alt=""
      draggable={false}
      className="pointer-events-none z-10 size-full object-fill [image-rendering:pixelated]"
      style={{
        gridColumn: `1 / span ${bounds.width}`,
        gridRow: `1 / span ${bounds.height}`,
      }}
    />
  ) : null;
}
