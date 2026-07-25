import type {
  MediaAsset,
  PresignedUploadRequest,
  PresignedUploadTarget,
} from "@/types/media";

export const mockMediaAssets: MediaAsset[] = [
  {
    id: "media-forager-reference",
    kind: "image",
    fileName: "forager-reference.png",
    contentType: "image/png",
    objectKey: "projects/moonlit-orchard/references/forager-reference.png",
  },
];

export function createMockPresignedUploadTarget(
  request: PresignedUploadRequest,
): PresignedUploadTarget {
  const mediaId = `media-${crypto.randomUUID()}`;
  const objectKey = `uploads/${mediaId}/${request.fileName}`;

  return {
    mediaId,
    objectKey,
    uploadUrl: `https://mock-storage.local/${objectKey}`,
    expiresAt: new Date(Date.now() + 15 * 60 * 1000).toISOString(),
  };
}
