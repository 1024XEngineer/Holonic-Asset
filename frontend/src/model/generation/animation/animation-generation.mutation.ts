import { useMutation, useQueryClient } from "@tanstack/react-query";

import { animationGenerationApi } from "./animation-generation.api";
import { readAuthenticatedUserId } from "@/model/auth";
import { generationKeys } from "../run/keys";
import type { GenerationRun } from "../run/types";

export function useGenerateAnimationMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: animationGenerationApi.generate,
    onSuccess: (run) => {
      queryClient.setQueryData<GenerationRun[]>(
        generationKeys.runs(
          readAuthenticatedUserId(),
          run.projectId,
          run.assetId,
        ),
        (current = []) => [
          ...current.filter((currentRun) => currentRun.id !== run.id),
          run,
        ],
      );
    },
  });
}
