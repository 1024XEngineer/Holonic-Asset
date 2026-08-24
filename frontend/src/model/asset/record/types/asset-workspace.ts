import type { AssetRevision } from "../../types";
import type { Perspective } from "@/model/project";
import type { AssetRecordForKind, AssetRecordKind } from "./asset-record";

export type AssetWorkspaceAsset<K extends AssetRecordKind = AssetRecordKind> = {
  id: string;
  projectId: string;
  kind: K;
  name: string;
  perspective: Perspective;
  version: string;
  history: AssetRevision[];
};

export type AssetWorkspaceDataForKind<K extends AssetRecordKind> = {
  projectName: string;
  asset: AssetWorkspaceAsset<K>;
  record: AssetRecordForKind<K>;
};

export type AssetWorkspaceData = AssetWorkspaceDataForKind<AssetRecordKind>;
