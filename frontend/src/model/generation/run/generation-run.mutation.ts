import { useMutation, useQueryClient } from "@tanstack/react-query";

import { generationApi, type GenerationInput } from "./generation.api";
import type { GenerationRun } from "@/features/generation/types";
import { generationKeys } from "./keys";

export function useEnqueueGenerationMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: GenerationInput) => generationApi.enqueue(input),
    onSuccess: (run) => {
      queryClient.setQueryData<GenerationRun[]>(
        generationKeys.runs(run.projectId),
        (current = []) => [
          ...current.filter((currentRun) => currentRun.id !== run.id),
          run,
        ],
      );
    },
  });
}
