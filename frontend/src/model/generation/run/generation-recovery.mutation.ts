import { useMutation, useQueryClient } from "@tanstack/react-query";

import { readAuthenticatedUserId } from "@/model/auth";

import { coreGenerationApi } from "./core-generation.api";
import { forgetGenerationRunMetadata } from "./generation.api";
import { generationKeys } from "./keys";
import type { GenerationRun } from "./types";

export type GenerationRecoveryInput = {
  projectId: string;
  runId: string;
};

export async function retryGenerationRun(input: GenerationRecoveryInput) {
  return coreGenerationApi.retry(positiveCoreRunId(input.runId));
}

export async function deleteGenerationRun(input: GenerationRecoveryInput) {
  return coreGenerationApi.delete(positiveCoreRunId(input.runId));
}

export function useRetryGenerationRunMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: retryGenerationRun,
    onSuccess: (_response, input) => {
      const userId = readAuthenticatedUserId();
      queryClient.setQueriesData<GenerationRun[]>(
        { queryKey: generationKeys.runs(userId, input.projectId) },
        (current) =>
          current?.map((run) =>
            run.id === input.runId
              ? { ...run, status: "pending", error: undefined }
              : run,
          ),
      );
    },
  });
}

export function useDeleteGenerationRunMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: deleteGenerationRun,
    onSuccess: (_response, input) => {
      const userId = readAuthenticatedUserId();
      queryClient.setQueriesData<GenerationRun[]>(
        { queryKey: generationKeys.runs(userId, input.projectId) },
        (current) => current?.filter((run) => run.id !== input.runId),
      );
      forgetGenerationRunMetadata(input.projectId, [input.runId]);
    },
  });
}

function positiveCoreRunId(value: string) {
  const id = Number(value);
  if (!Number.isSafeInteger(id) || id <= 0) {
    throw new Error("generation run requires a persisted Core API identifier.");
  }
  return id;
}
