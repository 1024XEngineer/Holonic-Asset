import { useMutation, useQueryClient } from "@tanstack/react-query";

import { animationGenerationApi } from "./animation-generation.api";
import { readAuthenticatedUserId } from "@/model/auth";
import { generationKeys } from "../run/keys";
import type { GenerationRun } from "../run/types";

export function useGenerateAnimationMutation() {
  return useAnimationGenerationMutation(animationGenerationApi.generate);
}

export function useDeriveAnimationMutation() {
  return useAnimationGenerationMutation(animationGenerationApi.derive);
}

function useAnimationGenerationMutation<
  Input extends
    | Parameters<typeof animationGenerationApi.generate>[0]
    | Parameters<typeof animationGenerationApi.derive>[0],
>(mutationFn: (input: Input) => Promise<GenerationRun>) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn,
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
