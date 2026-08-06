import type { AssetKind, ProjectAsset } from "../../types";

export type AssetGroup = { kind: AssetKind; assets: ProjectAsset[] };
export type AssetLibraryItem = ProjectAsset & { kind: AssetKind };
export type AssetGroupsByProject = Record<string, AssetGroup[]>;
