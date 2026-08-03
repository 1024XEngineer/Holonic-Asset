import type { AssetKind, ProjectAsset } from "../../types";

export type AssetGroup = { kind: AssetKind; assets: ProjectAsset[] };
export type AssetGroupsByProject = Record<string, AssetGroup[]>;
