import type {
  AssetDetailResponse,
  AssetRecordResponse,
  DeleteAssetRequest,
  DeleteAssetResponse,
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
import { coreApiClient, unwrapApiResponse } from "@/model/fetchers";

export const coreAssetApi = {
  list: async (projectID: number, query?: ListAssetsQuery) =>
    unwrapApiResponse<GetAssetsResponse>(
      await coreApiClient.GET("/projects/{project_id}/assets", {
        params: { path: { project_id: projectID }, query },
      }),
    ),
  detail: async (assetID: number) =>
    unwrapApiResponse<AssetDetailResponse>(
      await coreApiClient.GET("/asset/{asset_id}", {
        params: { path: { asset_id: assetID } },
      }),
    ),
  records: async (assetID: number) =>
    unwrapApiResponse<GetAssetRecordsResponse>(
      await coreApiClient.GET("/asset/{asset_id}/records", {
        params: { path: { asset_id: assetID } },
      }),
    ),
  record: async (request: RecordAssetRequest) =>
    unwrapApiResponse<AssetRecordResponse>(
      await coreApiClient.POST("/asset/save", { body: request }),
    ),
  copy: async (request: CopyAssetRequest) =>
    unwrapApiResponse<CopyAssetResponse>(
      await coreApiClient.POST("/asset/copy", { body: request }),
    ),
  rollback: async (request: RollBackAssetRequest) =>
    unwrapApiResponse<RollBackAssetResponse>(
      await coreApiClient.POST("/asset/rollback", { body: request }),
    ),
  update: async (request: UpdateAssetRequest) =>
    unwrapApiResponse<UpdateAssetResponse>(
      await coreApiClient.PUT("/asset/update", { body: request }),
    ),
  delete: async (request: DeleteAssetRequest) =>
    unwrapApiResponse<DeleteAssetResponse>(
      await coreApiClient.DELETE("/asset/delete", { body: request }),
    ),
};
