import { assetApi } from "@/features/assets/api";
import { listMockProjects } from "@/features/project/api/mock";
import { DataApiError } from "@/lib/data-api-error";
import {
  createDefaultEditorDocument,
  mergeEditorDocument,
} from "./record-defaults";
import { runMockRequest, type MockRequestOptions } from "@/lib/mock-request";
import {
  isEditorDocumentForAssetKind,
  type EditorWorkspaceData,
} from "../../domain";
import type {
  GetEditorDocumentInput,
  SaveEditorDocumentInput,
} from "../record.contract";

export function getMockEditorDocument(
  input: GetEditorDocumentInput,
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
    const fallback = createDefaultEditorDocument(group.kind, asset);

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
      content: mergeEditorDocument(
        group.kind,
        fallback,
        currentRevision?.content,
      ),
    } as EditorWorkspaceData;
  }, options);
}

export async function saveMockEditorDocumentRevision({
  projectId,
  assetId,
  content,
}: SaveEditorDocumentInput) {
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
  const documentMode = content.mode;
  if (!isEditorDocumentForAssetKind(assetGroup.kind, content)) {
    throw new DataApiError(
      "BAD_REQUEST",
      "Editor document does not match the asset kind.",
      { projectId, assetId, assetKind: assetGroup.kind, mode: documentMode },
    );
  }

  const updatedGroups = await assetApi.saveRevision({
    projectId,
    assetId,
    description: content.prompt,
    payload: content,
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
    content: structuredClone(content),
  };
}
