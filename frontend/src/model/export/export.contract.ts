import type { components, operations } from "@/model/generated/core-api";

export type CreateAssetExportRequest =
  operations["createAssetExport"]["requestBody"]["content"]["application/json"];
export type CreateAssetExportResponse = components["schemas"]["CreateResponse"];
export type AssetExportResponse = components["schemas"]["ExportResponse"];

export type AssetExportState =
  | { phase: "idle" }
  | { phase: "creating" }
  | { phase: "processing" }
  | { phase: "completed"; result: AssetExportResponse }
  | { phase: "failed"; message: string };
