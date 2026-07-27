import { createMockPresignedUploadTarget, mockMediaAssets } from "./mock/media";
import type { PresignedUploadRequest } from "@/features/media/domain";

export const mediaApi = {
  list: async () => structuredClone(mockMediaAssets),
  createUploadTarget: (request: PresignedUploadRequest) =>
    Promise.resolve(createMockPresignedUploadTarget(request)),
};
