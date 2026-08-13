import { coreAssetApi } from "./core-asset.api";
import type {
  AssetDetailResponse,
  AssetListItemResponse,
} from "./asset.contract";
import {
  assetCanvasSizeDimensionsSchema,
  resolveAssetCanvasSize,
} from "./asset-canvas-size";
import type { AssetKind, AssetMetadataUpdate, ProjectAsset } from "../types";

export { coreAssetApi } from "./core-asset.api";

export type SaveAssetRevisionInput<Payload> = {
  projectId: string;
  assetId: string;
  description: string;
  payload: Payload;
};

export const assetApi = {
  listGroups: async (projectId: string) => {
    const response = await coreAssetApi.list(coreProjectId(projectId));
    const details = await readAssetDetails(response.assets);
    return toAssetGroups(response.assets, new Map(details));
  },
  copy: async (projectId: string, assetId: string) => {
    await coreAssetApi.copy({ assetId: coreAssetId(assetId) });
    return assetApi.listGroups(projectId);
  },
  delete: async (assetId: string) => {
    return coreAssetApi.delete({ assetId: coreAssetId(assetId) });
  },
  update: async (
    projectId: string,
    assetId: string,
    metadata: AssetMetadataUpdate,
  ) => {
    const dimensions = parseAssetDimensions(metadata.canvasSize);
    await coreAssetApi.update({
      assetId: coreAssetId(assetId),
      name: metadata.name,
      description: metadata.description,
      tags: metadata.tags,
      perspective: metadata.perspective,
      ...(dimensions ? { dimensions } : {}),
    });
    return assetApi.listGroups(projectId);
  },
  saveRevision: <Payload>(_input: SaveAssetRevisionInput<Payload>) => {
    return Promise.reject(
      new Error(
        "Asset revisions cannot be saved through the Core API until the save payload is supported.",
      ),
    );
  },
};

const assetDetailConcurrency = 4;

async function readAssetDetails(assets: AssetListItemResponse[]) {
  const details: (readonly [number, AssetDetailResponse])[] = [];
  for (let index = 0; index < assets.length; index += assetDetailConcurrency) {
    const batch = assets.slice(index, index + assetDetailConcurrency);
    const results = await Promise.allSettled(
      batch.map(
        async (asset) =>
          [asset.assetId, await coreAssetApi.detail(asset.assetId)] as const,
      ),
    );
    for (const result of results) {
      if (result.status === "fulfilled") details.push(result.value);
    }
  }
  return details;
}

function parseAssetDimensions(canvasSize: string) {
  if (canvasSize.trim().toUpperCase() === "N/A") return undefined;
  const result = assetCanvasSizeDimensionsSchema.safeParse(canvasSize);
  if (!result.success) {
    throw new Error(result.error.issues[0]?.message ?? "Invalid canvas size.");
  }
  return result.data;
}

export function toAssetGroups(
  items: AssetListItemResponse[],
  details: ReadonlyMap<number, AssetDetailResponse> = new Map(),
) {
  const groups = new Map<AssetKind, ProjectAsset[]>();

  for (const item of items) {
    const kind = item.type === "tileSet" ? "tileset" : item.type;
    const assets = groups.get(kind) ?? [];
    const detail = details.get(item.assetId);
    const prototypeUrls = readPrototypeURLs(detail?.content);
    const thumbnailUrl = prototypeUrls?.[0];
    assets.push({
      id: String(item.assetId),
      name: item.name,
      description: item.description,
      version: `v${item.version}`,
      canvasSize: resolveAssetCanvasSize(item),
      perspective: item.perspective,
      tags: item.tags ?? [],
      ...(thumbnailUrl ? { thumbnailUrl } : {}),
      ...(prototypeUrls ? { prototypeUrls } : {}),
      history: [],
      animations: [],
    });
    groups.set(kind, assets);
  }

  return [...groups].map(([kind, assets]) => ({ kind, assets }));
}

function readPrototypeURLs(content: unknown) {
  if (!content || typeof content !== "object") return undefined;
  const prototype = (content as { prototype?: unknown }).prototype;
  if (!Array.isArray(prototype) || prototype.length === 0) return undefined;
  const urls = prototype.flatMap((resource) => {
    const url = (resource as { url?: unknown } | null)?.url;
    return typeof url === "string" && url.length > 0 ? [url] : [];
  });
  return urls.length > 0 ? urls : undefined;
}

function coreProjectId(projectId: string) {
  const value = Number(projectId);
  if (!Number.isSafeInteger(value) || value <= 0) {
    throw new Error("Asset loading requires a persisted Core API project.");
  }
  return value;
}

function coreAssetId(assetId: string) {
  const value = Number(assetId);
  if (!Number.isSafeInteger(value) || value <= 0) {
    throw new Error("Asset operations require a persisted Core API asset.");
  }
  return value;
}
