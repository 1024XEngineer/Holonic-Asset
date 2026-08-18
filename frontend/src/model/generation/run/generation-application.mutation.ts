import { useMutation, useQueryClient } from "@tanstack/react-query";

import { readAuthenticatedUserId } from "@/model/auth";
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
  await coreGenerationApi.resolveApplication(runId, input.applied);
}

export function useResolveGenerationApplicationMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: resolveGenerationApplication,
    onSuccess: async (_, input) => {
      const userId = readAuthenticatedUserId();
      await queryClient.invalidateQueries({
        queryKey: generationKeys.runs(userId, input.projectId, input.assetId),
      });
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
