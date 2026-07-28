import type { AssetKind, AssetRevision } from "@/features/assets/domain";
import type { EditorRecordForKind } from "./editor-record";

export type EditorWorkspaceAsset<K extends AssetKind = AssetKind> = {
  id: string;
  projectId: string;
  kind: K;
  name: string;
  version: string;
  history: AssetRevision[];
};

export type EditorWorkspaceDataForKind<K extends AssetKind> = {
  projectName: string;
  asset: EditorWorkspaceAsset<K>;
  record: EditorRecordForKind<K>;
};

export type EditorWorkspaceData = EditorWorkspaceDataForKind<AssetKind>;
