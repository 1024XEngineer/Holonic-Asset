import type { AssetKind } from "./asset-kind";
import type { ProjectAsset } from "./asset";

export type AssetGroup = { kind: AssetKind; assets: ProjectAsset[] };
export type AssetGroupsByProject = Record<string, AssetGroup[]>;
