import {
  addMockAsset,
  copyMockAsset,
  deleteMockAsset,
  listMockAssetGroups,
  saveMockAssetRevision,
} from "@/api/mock";
import type { ProjectAsset } from "@/domain/asset";
import type { AssetKind } from "@/domain/asset";
import type { RecordContent } from "@/domain/asset";

export const assetApi = {
  listGroups: (projectId: string) => listMockAssetGroups(projectId),
  add: (projectId: string, kind: AssetKind, asset: ProjectAsset) =>
    addMockAsset(projectId, kind, asset),
  copy: (projectId: string, assetId: string) =>
    copyMockAsset(projectId, assetId),
  delete: (projectId: string, assetId: string) =>
    deleteMockAsset(projectId, assetId),
  saveRevision: (projectId: string, assetId: string, content: RecordContent) =>
    saveMockAssetRevision(projectId, assetId, content),
};
