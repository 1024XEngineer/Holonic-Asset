import type { AssetRevision } from "../../types";
import type { AssetAttributes } from "../../types";
import type {
  AssetContentForKind,
  AssetContentKind,
} from "../../content/types";

export type AssetRecordState = {
  version: string;
  history: AssetRevision[];
};

export type AssetSnapshotForKind<K extends AssetContentKind> = {
  projectName: string;
  attributes: AssetAttributes<K>;
  content: AssetContentForKind<K>;
  record: AssetRecordState;
};

export type AssetSnapshot = AssetSnapshotForKind<AssetContentKind>;

/** @deprecated Use AssetAttributes and AssetRecordState. */
export type AssetWorkspaceAsset<K extends AssetContentKind = AssetContentKind> =
  {
    id: string;
    projectId: string;
    kind: K;
    name: string;
    perspective: AssetAttributes<K>["perspective"];
    version: string;
    history: AssetRevision[];
  };

/** @deprecated Use AssetSnapshot. */
export type AssetWorkspaceDataForKind<K extends AssetContentKind> = {
  projectName: string;
  asset: AssetWorkspaceAsset<K>;
  record: AssetContentForKind<K>;
};

/** @deprecated Use AssetSnapshot. */
export type AssetWorkspaceData = AssetWorkspaceDataForKind<AssetContentKind>;
