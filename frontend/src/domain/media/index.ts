export type MediaKind = "image" | "audio" | "archive";
export type PresignedUploadRequest = {
  fileName: string;
  contentType: string;
  contentLength: number;
  kind: MediaKind;
};
export type PresignedUploadTarget = {
  mediaId: string;
  objectKey: string;
  uploadUrl: string;
  expiresAt: string;
};
export type MediaAsset = {
  id: string;
  kind: MediaKind;
  fileName: string;
  contentType: string;
  objectKey: string;
};
