import { coreAssetApi } from "./core-asset.api";
import type { AssetListItemResponse } from "./asset.contract";
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
    return toAssetGroups(response.assets);
  },
  copy: async (projectId: string, assetId: string) => {
    await coreAssetApi.copy({ assetId: coreAssetId(assetId) });
    return assetApi.listGroups(projectId);
  },
  delete: async (projectId: string, assetId: string) => {
    await coreAssetApi.delete({ assetId: coreAssetId(assetId) });
    return assetApi.listGroups(projectId);
  },
  update: async (
    projectId: string,
    assetId: string,
    metadata: AssetMetadataUpdate,
  ) => {
    await coreAssetApi.update({
      assetId: coreAssetId(assetId),
      name: metadata.name,
      description: metadata.description,
      tags: metadata.tags,
      perspective: metadata.perspective,
      dimensions: assetCanvasSizeDimensionsSchema.parse(metadata.canvasSize),
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

export function toAssetGroups(items: AssetListItemResponse[]) {
  const groups = new Map<AssetKind, ProjectAsset[]>();

  for (const item of items) {
    const kind = item.type === "tileSet" ? "tileset" : item.type;
    const assets = groups.get(kind) ?? [];
    assets.push({
      id: String(item.assetId),
      name: item.name,
      description: item.description,
      version: `v${item.version}`,
      canvasSize: resolveAssetCanvasSize(item),
      perspective: item.perspective,
      tags: item.tags ?? [],
      history: [],
      animations: [],
    });
    groups.set(kind, assets);
  }

  return [...groups].map(([kind, assets]) => ({ kind, assets }));
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
