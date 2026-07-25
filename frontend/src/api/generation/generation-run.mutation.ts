import { useMutation, useQueryClient } from "@tanstack/react-query";

import { generationApi } from "./generation.api";
import type {
  GenerationInput,
  GenerationLifecycleUpdate,
} from "@/domain/generation";
import type { GenerationRun } from "@/domain/generation";
import { assetKeys } from "@/api/asset/keys";
import { generationKeys } from "./keys";

export function useEnqueueGenerationMutation() {
  const queryClient = useQueryClient();

  const projectUpdate = (update: GenerationLifecycleUpdate) => {
    const queryKey = generationKeys.runs(
      update.kind === "run-upserted" ? update.run.projectId : update.projectId,
    );
    if (
      update.kind === "run-removed" &&
      queryClient.getQueryData(queryKey) === undefined
    ) {
      return;
    }

    queryClient.setQueryData<GenerationRun[]>(queryKey, (current = []) =>
      update.kind === "run-upserted"
        ? [...current.filter((run) => run.id !== update.run.id), update.run]
        : current.filter((run) => run.id !== update.runId),
    );
  };

  return useMutation({
    mutationFn: (input: GenerationInput) =>
      generationApi.enqueue(input, projectUpdate),
    onSuccess: ({ assetGroups, run }) => {
      if (!assetGroups) return;
      queryClient.setQueryData(assetKeys.library(run.projectId), assetGroups);
    },
  });
}
