import type {
  AssetDetailResponse,
  AssetRecordResponse,
} from "../library/asset.contract";
import type { AssetRevision } from "../types";
import type { AssetRecord, AssetWorkspaceData } from "./types";

export function createAssetSnapshot({
  projectId,
  projectName,
  detail,
  kind,
  record,
  records,
}: {
  projectId: string;
  projectName: string;
  detail: AssetDetailResponse;
  kind: AssetWorkspaceData["asset"]["kind"];
  record: AssetRecord;
  records: AssetRecordResponse[];
}): AssetWorkspaceData {
  return {
    projectName,
    asset: {
      id: String(detail.assetId),
      projectId,
      kind,
      name: detail.name,
      perspective: detail.perspective,
      version: `v${detail.version}`,
      history: toAssetHistory(records, detail.version),
    },
    record,
  } as AssetWorkspaceData;
}

export function toAssetHistory(
  records: AssetRecordResponse[],
  currentVersion: number,
) {
  return records.map(
    (record): AssetRevision => ({
      id: String(record.recordId),
      version: `v${record.version}`,
      description: record.description,
      savedAt: record.createdAt,
      status: "ready",
      isCurrent: record.version === currentVersion,
    }),
  );
}
