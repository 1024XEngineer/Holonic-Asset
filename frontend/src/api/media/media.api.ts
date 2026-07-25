import { createMockPresignedUploadTarget, mockMediaAssets } from "@/api/mock";
import type { PresignedUploadRequest } from "@/types/media";

export const mediaApi = {
  list: async () => structuredClone(mockMediaAssets),
  createUploadTarget: (request: PresignedUploadRequest) =>
    Promise.resolve(createMockPresignedUploadTarget(request)),
};
