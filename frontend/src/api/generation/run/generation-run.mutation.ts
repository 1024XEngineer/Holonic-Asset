import { useMutation, useQueryClient } from "@tanstack/react-query";

import { generationApi } from "./generation.api";
import type { GenerationInput } from "@/model";
import type { GenerationRun } from "@/model";
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
