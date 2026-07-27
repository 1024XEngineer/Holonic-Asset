import {
  addMockAsset,
  copyMockAsset,
  deleteMockAsset,
  listMockAssetGroups,
} from "./mock";
import type { AssetKind, ProjectAsset } from "@/features/assets/domain";

export const assetApi = {
  listGroups: (projectId: string) => listMockAssetGroups(projectId),
  add: (projectId: string, kind: AssetKind, asset: ProjectAsset) =>
    addMockAsset(projectId, kind, asset),
  copy: (projectId: string, assetId: string) =>
    copyMockAsset(projectId, assetId),
  delete: (projectId: string, assetId: string) =>
    deleteMockAsset(projectId, assetId),
};
