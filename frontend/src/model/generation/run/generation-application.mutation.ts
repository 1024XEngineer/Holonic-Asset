import { useMutation, useQueryClient } from "@tanstack/react-query";

import { readAuthenticatedUserId } from "@/model/auth";
import { refreshAssetLibraryCache } from "../../asset/library/asset-library-cache";
import { coreAssetApi } from "../../asset/library/core-asset.api";
import { recordQueryOptions } from "../../asset/record/record.query";
import { coreGenerationApi } from "./core-generation.api";
import { generationKeys } from "./keys";

export type ResolveGenerationApplicationInput = {
  projectId: string;
  assetId: string;
  runId: string;
  applied: boolean;
};

export async function resolveGenerationApplication(
  input: ResolveGenerationApplicationInput,
) {
  const runId = positiveCoreId(input.runId, "generation run");
  if (input.applied) {
    const assetId = positiveCoreId(input.assetId, "asset");
    const detail = await coreGenerationApi.detail(runId);
    if (detail.status !== "awaiting_application") {
      throw new Error("Generation result is no longer awaiting application.");
    }
    if (
      detail.result?.asset_id !== assetId ||
      detail.result.version === undefined ||
      detail.result.content === undefined
    ) {
      throw new Error(
        "Generation result does not contain an applicable asset revision.",
      );
    }
    await coreAssetApi.record({
      assetId,
      expectedVersion: detail.result.version,
      content: detail.result.content,
    });
  }

  await coreGenerationApi.resolveApplication(runId, input.applied);
}

export function useResolveGenerationApplicationMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: resolveGenerationApplication,
    onSuccess: async (_, input) => {
      const userId = readAuthenticatedUserId();
      const refreshes: Promise<unknown>[] = [
        queryClient.invalidateQueries({
          queryKey: generationKeys.runs(userId, input.projectId, input.assetId),
        }),
      ];
      if (input.applied) {
        refreshes.push(
          refreshAssetLibraryCache(queryClient, userId, input.projectId),
          queryClient.refetchQueries(
            {
              queryKey: recordQueryOptions(input.projectId, input.assetId)
                .queryKey,
              type: "all",
            },
            { throwOnError: true },
          ),
        );
      }
      await Promise.all(refreshes);
    },
  });
}

function positiveCoreId(value: string, resource: string) {
  const id = Number(value);
  if (!Number.isSafeInteger(id) || id <= 0) {
    throw new Error(`${resource} requires a persisted Core API identifier.`);
  }
  return id;
}
