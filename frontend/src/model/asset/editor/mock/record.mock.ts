import { assetApi } from "../../library/asset.api";
import { listMockProjects } from "../../../project/mock";
import { DataApiError } from "@/lib/data-api-error";
import {
  createDefaultEditorRecord,
  mergeEditorRecord,
} from "./record-defaults";
import { runMockRequest, type MockRequestOptions } from "@/lib/mock-request";
import {
  isEditorRecordForAssetKind,
  type EditorWorkspaceData,
} from "@/features/asset-editor/types";
import type {
  GetEditorRecordInput,
  SaveEditorRecordInput,
} from "../record.contract";

export function getMockEditorRecord(
  input: GetEditorRecordInput,
  options?: MockRequestOptions,
) {
  return runMockRequest(async (): Promise<EditorWorkspaceData> => {
    const [projects, groups] = await Promise.all([
      listMockProjects(),
      assetApi.listGroups(input.projectId),
    ]);
    const project = projects.find((item) => item.id === input.projectId);
    if (!project) {
      throw new DataApiError("NOT_FOUND", "Project was not found.", {
        projectId: input.projectId,
      });
    }

    const group = groups.find((item) =>
      item.assets.some((asset) => asset.id === input.assetId),
    );
    const asset = group?.assets.find((item) => item.id === input.assetId);
    if (!group || !asset) {
      throw new DataApiError("NOT_FOUND", "Asset was not found.", input);
    }

    const currentRevision = asset.history.find(
      (revision) => revision.isCurrent,
    );
    const fallback = createDefaultEditorRecord(group.kind, asset);

    return {
      projectName: project.name,
      asset: {
        id: asset.id,
        projectId: input.projectId,
        kind: group.kind,
        name: asset.name,
        version: asset.version,
        history: structuredClone(asset.history),
      },
      record: mergeEditorRecord(group.kind, fallback, currentRevision?.content),
    } as EditorWorkspaceData;
  }, options);
}

export async function saveMockEditorRecordRevision({
  projectId,
  assetId,
  record,
}: SaveEditorRecordInput) {
  const groups = await assetApi.listGroups(projectId);
  const assetGroup = groups.find((group) =>
    group.assets.some((asset) => asset.id === assetId),
  );
  if (!assetGroup) {
    throw new DataApiError("NOT_FOUND", "Asset was not found.", {
      projectId,
      assetId,
    });
  }
  const recordMode = record.mode;
  if (!isEditorRecordForAssetKind(assetGroup.kind, record)) {
    throw new DataApiError(
      "BAD_REQUEST",
      "Editor record does not match the asset kind.",
      { projectId, assetId, assetKind: assetGroup.kind, mode: recordMode },
    );
  }

  const updatedGroups = await assetApi.saveRevision({
    projectId,
    assetId,
    description: record.prompt,
    payload: record,
  });
  const savedAsset = updatedGroups
    .flatMap((group) => group.assets)
    .find((asset) => asset.id === assetId);
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
