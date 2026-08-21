import type {
  CreateUploadTargetRequest,
  UploadTarget,
} from "./upload.contract";
import { coreApiClient, unwrapApiResponse } from "@/model/fetchers";

export type {
  CreateUploadTargetRequest,
  UploadTarget,
} from "./upload.contract";

export const uploadApi = {
  createTarget: async (
    request: CreateUploadTargetRequest,
    signal?: AbortSignal,
  ) =>
    unwrapApiResponse<UploadTarget>(
      await coreApiClient.POST("/uploads", { body: request, signal }),
    ),
};
