import { listMockAssetGroups, listMockProjects } from "./workspace";
import { DataApiError } from "@/api/api-error";
import { createDefaultRecord, mergeRecord } from "./record-defaults";
import { runMockRequest, type MockRequestOptions } from "./request";
import type { RecordData } from "@/types/record";

export type GetRecordInput = {
  projectId: string;
  assetId: string;
};

export function getMockRecord(
  input: GetRecordInput,
  options?: MockRequestOptions,
) {
  return runMockRequest(async (): Promise<RecordData> => {
    const [projects, groups] = await Promise.all([
      listMockProjects(),
      listMockAssetGroups(input.projectId),
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

    const currentRecord = asset.history.find((record) => record.isCurrent);
    const fallback = createDefaultRecord(group.kind, asset);

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
      content: mergeRecord(group.kind, fallback, currentRecord?.content),
    } as RecordData;
  }, options);
}
