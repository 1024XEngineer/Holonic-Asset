import { assetApi } from "../../library/asset.api";
import { createAssetLibraryCollection } from "../../library/asset-library-collection";
import { projectApi } from "@/model/project";
import { DataApiError } from "@/lib/data-api-error";
import { createDefaultAssetRecord, mergeAssetRecord } from "./record-defaults";
import { runMockRequest, type MockRequestOptions } from "@/lib/mock-request";
import type { AssetRecord, AssetWorkspaceData } from "../types";
import type { GetAssetRecordInput, SaveAssetRecordInput } from "../types";

export function getMockAssetRecord(
  input: GetAssetRecordInput,
  options?: MockRequestOptions,
) {
  return runMockRequest(async (): Promise<AssetWorkspaceData> => {
    const [project, groups] = await Promise.all([
      projectApi.detail(input.projectId),
      assetApi.listGroups(input.projectId),
    ]);

    const asset = createAssetLibraryCollection(groups).find(input.assetId);
    if (!asset) {
      throw new DataApiError("NOT_FOUND", "Asset was not found.", input);
    }

    const currentRevision = asset.history.find(
      (revision) => revision.isCurrent,
    );
    const fallback = createDefaultAssetRecord(asset.kind, asset);

    return {
      projectName: project.name,
      asset: {
        id: asset.id,
        projectId: input.projectId,
        kind: asset.kind,
        name: asset.name,
        perspective: asset.perspective,
        version: asset.version,
        history: structuredClone(asset.history),
      },
      record: mergeAssetRecord(
        asset.kind,
        fallback,
        currentRevision?.content as AssetRecord | undefined,
      ),
    } as AssetWorkspaceData;
  }, options);
}

export async function saveMockAssetRecordRevision({
  projectId,
  assetId,
  record,
}: SaveAssetRecordInput) {
  const groups = await assetApi.listGroups(projectId);
  const asset = createAssetLibraryCollection(groups).find(assetId);
  if (!asset) {
    throw new DataApiError("NOT_FOUND", "Asset was not found.", {
      projectId,
      assetId,
    });
  }
  const updatedGroups = await assetApi.saveRevision({
    projectId,
    assetId,
    description: record.prompt,
    payload: record,
  });
  const savedAsset = createAssetLibraryCollection(updatedGroups).find(assetId);
  if (!savedAsset) {
    throw new DataApiError("UNKNOWN", "Saved asset could not be reloaded.", {
      projectId,
      assetId,
    });
  }

  return {
    projectId,
    assetId,
    version: savedAsset.version,
    history: structuredClone(savedAsset.history),
    record: structuredClone(record),
  };
}
