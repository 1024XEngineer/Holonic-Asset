import { useCallback, useEffect, useRef, useState } from "react";
import type { MutableRefObject } from "react";

import { coreExportApi } from "./export.api";
import type { AssetExportResponse, AssetExportState } from "./export.contract";

const EXPORT_POLL_INTERVAL_MS = 1_000;

export function useAssetExport() {
  const [state, setState] = useState<AssetExportState>({ phase: "idle" });
  const operationRef = useRef(0);

  const exportAsset = useCallback(async (assetId: number) => {
    const operation = ++operationRef.current;
    setState({ phase: "creating" });

    try {
      const created = await coreExportApi.create({ assetId });
      if (operation !== operationRef.current) return;
      setState({ phase: "processing" });

      const result = await waitForExport(
        created.exportId,
        operation,
        operationRef,
      );
      if (operation !== operationRef.current) return;
      downloadExport(result);
      setState({ phase: "completed", result });
    } catch (error) {
      if (operation !== operationRef.current) return;
      setState({ phase: "failed", message: getExportErrorMessage(error) });
      throw error;
    }
  }, []);

  useEffect(
    () => () => {
      operationRef.current += 1;
    },
    [],
  );

  return {
    exportAsset,
    isExporting: state.phase === "creating" || state.phase === "processing",
    state,
  };
}

async function waitForExport(
  exportId: number,
  operation: number,
  operationRef: MutableRefObject<number>,
) {
  while (true) {
    await delay(EXPORT_POLL_INTERVAL_MS);
    if (operation !== operationRef.current)
      throw new Error("Export cancelled.");

    const result = await coreExportApi.get(exportId);
    if (result.status === "completed") {
      if (!result.downloadUrl)
        throw new Error("Export download is unavailable.");
      return result;
    }
    if (result.status === "failed" || result.status === "cancelled") {
      throw new Error(result.error || "Export failed.");
    }
  }
}

function downloadExport(result: AssetExportResponse) {
  const link = document.createElement("a");
  link.href = result.downloadUrl!;
  link.download = result.fileName || "asset-export.zip";
  link.rel = "noopener";
  document.body.appendChild(link);
  link.click();
  link.remove();
}

function delay(milliseconds: number) {
  return new Promise<void>((resolve) => setTimeout(resolve, milliseconds));
}

function getExportErrorMessage(error: unknown) {
  return error instanceof Error ? error.message : "Export failed.";
}
