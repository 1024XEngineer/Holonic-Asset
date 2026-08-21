import { uploadApi } from "./upload.api";
import type { UploadTarget } from "./upload.contract";

export async function uploadFile(
  file: File,
  signal?: AbortSignal,
): Promise<UploadTarget> {
  if (!file.type) {
    throw new Error("A concrete file content type is required.");
  }

  const target = await uploadApi.createTarget(
    {
      contentType: file.type,
      contentLength: file.size,
    },
    signal,
  );
  if (
    !target.uploadURL ||
    !target.uploadToken ||
    !target.objectKey ||
    !target.objectURL
  ) {
    throw new Error("The API returned an invalid upload target.");
  }

  const body = new FormData();
  body.append("token", target.uploadToken);
  body.append("key", target.objectKey);
  body.append("file", file, file.name);

  const response = await fetch(target.uploadURL, {
    method: "POST",
    body,
    signal,
  });
  if (!response.ok) {
    throw new Error(`File upload failed (${response.status}).`);
  }

  return target;
}
