import { postJson } from "@/api/fetchers";

/** Matches core-api/internal/dto.CreateUploadTargetRequest. */
export type CreateUploadTargetRequest = {
  contentType: string;
  contentLength: number;
};

/** Matches core-api/internal/dto.UploadTarget. */
export type UploadTarget = {
  objectKey: string;
  objectURL: string;
  uploadURL: string;
  uploadToken: string;
};

export const uploadApi = {
  createTarget: (request: CreateUploadTargetRequest) =>
    postJson<UploadTarget>("/uploads", request),
};
