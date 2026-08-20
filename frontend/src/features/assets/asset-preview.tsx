import { ImageOff, LoaderCircle } from "lucide-react";
import { useTranslation } from "react-i18next";

import { AssetKindIcon, getAssetKindConfig } from "@/components/asset-kind";
import { cn } from "@/lib/utils";
import { getGridBounds } from "@/lib/grid-bounds";
import {
  useRecordQuery,
  type AssetRecordForKind,
  type AssetLibraryItem,
  type SceneryLayer,
} from "@/model/asset";

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

type AssetPreviewState =
  | { value: "tileset"; assetId: string; projectId: string }
  | { value: "scenery"; assetId: string; name: string; projectId: string }
  | {
      value: "crop";
      name: string;
      thumbnailUrl: string;
      previewCrop: NonNullable<AssetPreviewAsset["previewCrop"]>;
    }
  | {
      value: "frame";
      name: string;
      thumbnailUrl: string;
      previewFrame: NonNullable<AssetPreviewAsset["previewFrame"]>;
    }
  | {
      value: "image";
      name: string;
      thumbnailUrl: string;
      previewOffset?: AssetPreviewAsset["previewOffset"];
      previewScale?: number;
    }
  | { value: "unavailable"; kind: AssetPreviewAsset["kind"] };

type RecordPreviewState<K extends "scenery" | "tileset"> =
  | { value: "loading" }
  | { value: "ready"; record: AssetRecordForKind<K> }
  | { value: "unavailable" };

export function AssetPreview({
  asset,
  className,
  projectId,
}: {
  asset: AssetPreviewAsset;
  className?: string;
  projectId?: string;
}) {
  const state = resolveAssetPreviewState(asset, projectId);

  return (
    <div
      className={cn(
        "relative grid aspect-[4/3] place-items-center overflow-hidden bg-muted/70",
        className,
      )}
    >
      <AssetPreviewContent state={state} />
      <div
        aria-hidden="true"
        className="pointer-events-none absolute inset-0 ring-1 ring-inset ring-foreground/5"
      />
    </div>
  );
}

function resolveAssetPreviewState(
  asset: AssetPreviewAsset,
  projectId?: string,
): AssetPreviewState {
  if (projectId) {
    switch (asset.kind) {
      case "tileset":
        return { value: "tileset", assetId: asset.id, projectId };
      case "scenery":
        return {
          value: "scenery",
          assetId: asset.id,
          name: asset.name,
          projectId,
        };
    }
  }

  if (!asset.thumbnailUrl) return { value: "unavailable", kind: asset.kind };
  if (asset.previewCrop) {
    return {
      value: "crop",
      name: asset.name,
      thumbnailUrl: asset.thumbnailUrl,
      previewCrop: asset.previewCrop,
    };
  }
  if (asset.previewFrame) {
    return {
      value: "frame",
      name: asset.name,
      thumbnailUrl: asset.thumbnailUrl,
      previewFrame: asset.previewFrame,
    };
  }
  return {
    value: "image",
    name: asset.name,
    thumbnailUrl: asset.thumbnailUrl,
    ...(asset.previewOffset ? { previewOffset: asset.previewOffset } : {}),
    ...(asset.previewScale === undefined
      ? {}
      : { previewScale: asset.previewScale }),
  };
}

function AssetPreviewContent({ state }: { state: AssetPreviewState }) {
  switch (state.value) {
    case "tileset":
      return (
        <TilesetPreview assetId={state.assetId} projectId={state.projectId} />
      );
    case "scenery":
      return (
        <SceneryPreview
          assetId={state.assetId}
          name={state.name}
          projectId={state.projectId}
        />
      );
    case "crop": {
      const { name, previewCrop, thumbnailUrl } = state;
      return (
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
      );
    }
    case "frame": {
      const { name, previewFrame, thumbnailUrl } = state;
      return (
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
      );
    }
    case "image": {
      const { name, previewOffset, previewScale, thumbnailUrl } = state;
      return (
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
      );
    }
    case "unavailable":
      return (
        <div className="grid place-items-center gap-3 text-muted-foreground">
          <span
            className={cn(
              "grid size-14 place-items-center rounded-md text-white shadow-sm",
              getAssetKindConfig(state.kind).accentClassName,
            )}
          >
            <AssetKindIcon kind={state.kind} className="size-6" />
          </span>
          <span className="flex items-center gap-1.5 text-xs font-medium">
            <ImageOff className="size-3.5" />
            Preview unavailable
          </span>
        </div>
      );
  }
}

const sceneryLayerBlendClasses: Record<SceneryLayer["blendMode"], string> = {
  normal: "",
  multiply: "mix-blend-multiply",
};

function SceneryPreview({
  assetId,
  name,
  projectId,
}: {
  assetId: string;
  name: string;
  projectId: string;
}) {
  const state = useRecordPreviewState<"scenery">(projectId, assetId);

  switch (state.value) {
    case "loading":
      return <RecordPreviewStatus loading />;
    case "unavailable":
      return <RecordPreviewStatus />;
    case "ready": {
      const { dimensions = { width: 16, height: 9 }, layers } =
        state.record.scenery;

      return (
        <div
          aria-label={`${name} preview`}
          className="relative w-full max-h-full overflow-hidden bg-[#c8e8ed]"
          style={{ aspectRatio: `${dimensions.width} / ${dimensions.height}` }}
        >
          {layers.flatMap((layer) => {
            if (layer.visible === false) return [];

            const position = layer.position ?? { x: 0, y: 0 };
            const scale = layer.transform?.scale ?? { x: 1, y: 1 };
            const rotation = layer.transform?.rotation ?? 0;

            return [
              <img
                key={layer.id}
                alt=""
                className={cn(
                  "absolute object-cover",
                  sceneryLayerBlendClasses[layer.blendMode],
                )}
                loading="lazy"
                src={layer.imageUrl}
                style={{
                  left: `${(position.x / dimensions.width) * 100}%`,
                  top: `${(position.y / dimensions.height) * 100}%`,
                  width: `${scale.x * 100}%`,
                  height: `${scale.y * 100}%`,
                  opacity: layer.opacity ?? 1,
                  zIndex: layer.zIndex,
                  transform: `rotate(${rotation}deg)`,
                  transformOrigin: "center",
                }}
              />,
            ];
          })}
        </div>
      );
    }
  }
}

function TilesetPreview({
  assetId,
  projectId,
}: {
  assetId: string;
  projectId: string;
}) {
  const { t } = useTranslation("assets");
  const state = useRecordPreviewState<"tileset">(projectId, assetId);

  switch (state.value) {
    case "loading":
      return <RecordPreviewStatus loading />;
    case "unavailable":
      return <RecordPreviewStatus />;
    case "ready": {
      const { gridSize, items } = state.record.tileset;

      return (
        <div
          aria-label={t("tilesetPreview")}
          className="grid size-full overflow-hidden bg-[#eeece7] p-3"
          style={{
            gridTemplateColumns: `repeat(${gridSize}, minmax(0, 1fr))`,
            gridTemplateRows: `repeat(${gridSize}, minmax(0, 1fr))`,
          }}
        >
          {items.map((item) => {
            if (item.tiles.length === 0) return null;

            if (item.tileUrls) {
              return item.tiles.flatMap((tile, tileIndex) => {
                const url = item.tileUrls?.[tileIndex];
                if (!url) return [];

                return [
                  <img
                    key={`${item.id}:${tile[0]}:${tile[1]}`}
                    alt=""
                    className="z-10 size-full object-fill [image-rendering:pixelated]"
                    src={url}
                    style={{
                      gridColumn: tile[0] + 1,
                      gridRow: tile[1] + 1,
                    }}
                  />,
                ];
              });
            }

            if (!item.imageUrl) return null;

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
  }
}

function useRecordPreviewState<K extends "scenery" | "tileset">(
  projectId: string,
  assetId: string,
): RecordPreviewState<K> {
  const query = useRecordQuery(projectId, assetId);

  switch (query.status) {
    case "pending":
      return { value: "loading" };
    case "error":
      return { value: "unavailable" };
    case "success":
      return query.data
        ? {
            value: "ready",
            record: query.data.record as AssetRecordForKind<K>,
          }
        : { value: "unavailable" };
  }
}

function RecordPreviewStatus({ loading = false }: { loading?: boolean }) {
  const { t } = useTranslation("assets");

  return (
    <div className="grid size-full place-items-center bg-[#eeece7] text-[#47656a]">
      {loading ? (
        <LoaderCircle className="size-6 animate-spin" aria-hidden="true" />
      ) : (
        <div className="grid place-items-center gap-2">
          <ImageOff className="size-6" aria-hidden="true" />
          <span className="text-xs font-medium">{t("previewUnavailable")}</span>
        </div>
      )}
    </div>
  );
}
