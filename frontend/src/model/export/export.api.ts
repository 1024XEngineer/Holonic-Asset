import { coreApiClient, unwrapApiResponse } from "@/model/fetchers";

import type {
  AssetExportResponse,
  CreateAssetExportRequest,
  CreateAssetExportResponse,
} from "./export.contract";

export const coreExportApi = {
  create: async (request: CreateAssetExportRequest) =>
    unwrapApiResponse<CreateAssetExportResponse>(
      await coreApiClient.POST("/asset/export", { body: request }),
    ),
  get: async (exportID: number) =>
    unwrapApiResponse<AssetExportResponse>(
      await coreApiClient.GET("/export/{export_id}", {
        params: { path: { export_id: exportID } },
        cache: "no-store",
      }),
    ),
};
