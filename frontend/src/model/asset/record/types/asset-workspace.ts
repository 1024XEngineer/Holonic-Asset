import type { AssetKind, AssetRevision } from "../../types";
import type { AssetRecordForKind } from "./asset-record";

export type AssetWorkspaceAsset<K extends AssetKind = AssetKind> = {
  id: string;
  projectId: string;
  kind: K;
  name: string;
  version: string;
  history: AssetRevision[];
};

export type AssetWorkspaceDataForKind<K extends AssetKind> = {
  projectName: string;
  asset: AssetWorkspaceAsset<K>;
  record: AssetRecordForKind<K>;
};

export type AssetWorkspaceData = AssetWorkspaceDataForKind<AssetKind>;
