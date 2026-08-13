import { useMutation, useQueryClient } from "@tanstack/react-query";

import { generationApi } from "./generation.api";
import type { GenerationInput, GenerationRun } from "./types";
import { generationKeys } from "./keys";
import { readAuthenticatedUserId } from "@/model/auth";

export function useEnqueueGenerationMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: GenerationInput) => generationApi.enqueue(input),
    onSuccess: (run) => {
      const userID = readAuthenticatedUserId();
      queryClient.setQueryData<GenerationRun[]>(
        generationKeys.runs(userID, run.projectId),
        (current = []) => [
          ...current.filter((currentRun) => currentRun.id !== run.id),
          run,
        ],
      );
    },
  });
}
