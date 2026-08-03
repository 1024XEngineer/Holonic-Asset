import type {
  AssetDetailResponse,
  AssetRecordResponse,
  CopyAssetRequest,
  CopyAssetResponse,
  GetAssetRecordsResponse,
  GetAssetsResponse,
  ListAssetsQuery,
  RecordAssetRequest,
  RollBackAssetRequest,
  RollBackAssetResponse,
  UpdateAssetRequest,
  UpdateAssetResponse,
} from "./asset.contract";
import { getEnvelope, postEnvelope } from "@/model/fetchers";

export const coreAssetApi = {
  list: (projectID: number, query?: ListAssetsQuery) =>
    getEnvelope<GetAssetsResponse>(
      `/projects/${projectID}/assets`,
      query ? { ...query } : undefined,
    ),
  detail: (assetID: number) =>
    getEnvelope<AssetDetailResponse>(`/asset/${assetID}`),
  records: (assetID: number) =>
    getEnvelope<GetAssetRecordsResponse>(`/asset/${assetID}/records`),
  record: (request: RecordAssetRequest) =>
    postEnvelope<AssetRecordResponse>("/asset/save", request),
  copy: (request: CopyAssetRequest) =>
    postEnvelope<CopyAssetResponse>("/asset/copy", request),
  rollback: (request: RollBackAssetRequest) =>
    postEnvelope<RollBackAssetResponse>("/asset/rollback", request),
  update: (request: UpdateAssetRequest) =>
    postEnvelope<UpdateAssetResponse>("/asset/update", request),
};
