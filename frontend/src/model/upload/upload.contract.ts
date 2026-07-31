import type { operations } from "@/model/generated/core-api";

export type CreateUploadTargetRequest =
  operations["createUploadTarget"]["requestBody"]["content"]["application/json"];
export type UploadTarget =
  operations["createUploadTarget"]["responses"][200]["content"]["application/json"];
