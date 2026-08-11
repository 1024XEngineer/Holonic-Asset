import {
  addMockAsset,
  copyMockAsset,
  deleteMockAsset,
  listMockAssetGroups,
  saveMockAssetRevision,
  updateMockAsset,
} from "./mock";
import type { AssetListItemResponse } from "./asset.contract";
import { resolveAssetCanvasSize } from "./asset-canvas-size";
import type { AssetKind, AssetMetadataUpdate, ProjectAsset } from "../types";

export { coreAssetApi } from "./core-asset.api";

export type SaveAssetRevisionInput<Payload> = {
  projectId: string;
  assetId: string;
  description: string;
  payload: Payload;
};

export const assetApi = {
  listGroups: (projectId: string) => listMockAssetGroups(projectId),
  add: (projectId: string, kind: AssetKind, asset: ProjectAsset) =>
    addMockAsset(projectId, kind, asset),
  copy: (projectId: string, assetId: string) =>
    copyMockAsset(projectId, assetId),
  delete: (projectId: string, assetId: string) =>
    deleteMockAsset(projectId, assetId),
  update: (projectId: string, assetId: string, metadata: AssetMetadataUpdate) =>
    updateMockAsset(projectId, assetId, metadata),
  saveRevision: <Payload>({
    projectId,
    assetId,
    description,
    payload,
  }: SaveAssetRevisionInput<Payload>) =>
    saveMockAssetRevision(projectId, assetId, description, payload),
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
