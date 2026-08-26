import type { AssetExportResponse } from "@/model";

export function downloadAssetExport(result: AssetExportResponse) {
  if (!result.downloadUrl) return;

  const link = document.createElement("a");
  link.href = result.downloadUrl;
  link.download = result.fileName || "asset-export.zip";
  link.rel = "noopener";
  document.body.appendChild(link);
  link.click();
  link.remove();
}
