import { ImageOff } from "lucide-react";

import { AssetKindIcon, getAssetKindConfig } from "@/components/asset-kind";
import { cn } from "@/lib/utils";
import { getGridBounds } from "@/lib/grid-bounds";
import { useRecordQuery, type AssetLibraryItem } from "@/model/asset";

type AssetPreviewAsset = Pick<
  AssetLibraryItem,
  | "id"
  | "kind"
  | "name"
  | "previewCrop"
  | "previewFrame"
  | "previewOffset"
  | "previewScale"
  | "thumbnailUrl"
>;

export function AssetPreview({
  asset,
  className,
  projectId,
}: {
  asset: AssetPreviewAsset;
  className?: string;
  projectId?: string;
}) {
  const {
    id: assetId,
    kind,
    name,
    previewCrop,
    previewFrame,
    previewOffset,
    previewScale,
    thumbnailUrl,
  } = asset;

  return (
    <div
      className={cn(
        "relative grid aspect-[4/3] place-items-center overflow-hidden bg-muted/70",
        className,
      )}
    >
      {kind === "tileset" && projectId ? (
        <TilesetPreview assetId={assetId} projectId={projectId} />
      ) : thumbnailUrl && previewCrop ? (
        <div
          className="relative h-full max-w-full overflow-hidden"
          style={{
            aspectRatio: `${previewCrop.width} / ${previewCrop.height}`,
            transform: previewCrop.displayOffsetY
              ? `translateY(${previewCrop.displayOffsetY})`
              : undefined,
          }}
        >
          <img
            alt={`${name} preview`}
            className="absolute max-w-none [image-rendering:pixelated]"
            loading="lazy"
            src={thumbnailUrl}
            style={{
              height: `${(previewCrop.sourceHeight / previewCrop.height) * 100}%`,
              left: `-${(previewCrop.x / previewCrop.width) * 100}%`,
              top: `-${(previewCrop.y / previewCrop.height) * 100}%`,
            }}
          />
        </div>
      ) : thumbnailUrl && previewFrame ? (
        <div
          className={cn(
            "relative overflow-hidden",
            previewFrame.frameWidth && previewFrame.frameHeight
              ? "max-h-full"
              : "size-full",
          )}
          style={
            previewFrame.frameWidth && previewFrame.frameHeight
              ? {
                  aspectRatio: `${previewFrame.frameWidth} / ${previewFrame.frameHeight}`,
                  width: previewFrame.displayWidth ?? "100%",
                }
              : undefined
          }
        >
          <img
            alt={`${name} preview`}
            className="absolute top-1/2 left-1/2 max-w-none [image-rendering:pixelated]"
            loading="lazy"
            src={thumbnailUrl}
            style={{
              height: `${previewFrame.rows * 100}%`,
              transform: `translate(-${((previewFrame.column + 0.5) / previewFrame.columns) * 100}%, -${((previewFrame.row + 0.5) / previewFrame.rows) * 100}%) translateX(${previewFrame.offsetX ?? 0}px)`,
            }}
          />
        </div>
      ) : thumbnailUrl ? (
        <img
          alt={`${name} preview`}
          className="size-full object-contain p-5 [image-rendering:pixelated]"
          loading="lazy"
          src={thumbnailUrl}
          style={
            previewOffset || previewScale !== undefined
              ? {
                  transform: `translate(${previewOffset?.x ?? "0"}, ${previewOffset?.y ?? "0"}) scale(${previewScale ?? 1})`,
                }
              : undefined
          }
        />
      ) : (
        <div className="grid place-items-center gap-3 text-muted-foreground">
          <span
            className={cn(
              "grid size-14 place-items-center rounded-md text-white shadow-sm",
              getAssetKindConfig(kind).accentClassName,
            )}
          >
            <AssetKindIcon kind={kind} className="size-6" />
          </span>
          <span className="flex items-center gap-1.5 text-xs font-medium">
            <ImageOff className="size-3.5" />
            Preview unavailable
          </span>
        </div>
      )}
      <div
        aria-hidden="true"
        className="pointer-events-none absolute inset-0 ring-1 ring-inset ring-foreground/5"
      />
    </div>
  );
}

function TilesetPreview({
  assetId,
  projectId,
}: {
  assetId: string;
  projectId: string;
}) {
  const recordQuery = useRecordQuery(projectId, assetId);
  const record = recordQuery.data?.record;

  if (record?.mode !== "tileset") return <TilesetPreviewPlaceholder />;

  const { gridSize, items } = record.tileset;

  return (
    <div
      aria-label="Tileset preview"
      className="grid size-full overflow-hidden bg-[#eeece7] p-3"
      style={{
        gridTemplateColumns: `repeat(${gridSize}, minmax(0, 1fr))`,
        gridTemplateRows: `repeat(${gridSize}, minmax(0, 1fr))`,
      }}
    >
      {items.map((item) => {
        if (!item.imageUrl || item.tiles.length === 0) return null;

        const bounds = getGridBounds(item.tiles);

        return (
          <img
            key={item.id}
            alt=""
            className="z-10 size-full object-fill [image-rendering:pixelated]"
            src={item.imageUrl}
            style={{
              gridColumn: `${bounds.x + 1} / span ${bounds.width}`,
              gridRow: `${bounds.y + 1} / span ${bounds.height}`,
            }}
          />
        );
      })}
      {Array.from({ length: gridSize * gridSize }, (_, index) => (
        <span
          key={index}
          aria-hidden="true"
          className="z-20 border border-[#5dabb0]/65"
          style={{
            gridColumn: (index % gridSize) + 1,
            gridRow: Math.floor(index / gridSize) + 1,
          }}
        />
      ))}
    </div>
  );
}

function TilesetPreviewPlaceholder() {
  return (
    <div className="grid size-full grid-cols-8 grid-rows-8 bg-[#eeece7] p-3">
      {Array.from({ length: 64 }, (_, index) => (
        <span
          key={index}
          className={cn(
            "border border-[#5dabb0]/65",
            index % 5 === 0 ? "bg-emerald-500/20" : "bg-white/50",
          )}
        />
      ))}
    </div>
  );
}
