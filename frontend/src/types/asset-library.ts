import type { ProjectAsset } from "@/types/asset";
import type { AssetKind } from "@/types/asset-kind";

export type AssetGroup = {
  kind: AssetKind;
  assets: ProjectAsset[];
};

export type AssetGroupsByProject = Record<string, AssetGroup[]>;
