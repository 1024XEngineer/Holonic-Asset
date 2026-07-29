import { useMutation } from "@tanstack/react-query";

import { animationGenerationApi } from "./animation-generation.api";

export function useGenerateAnimationMutation() {
  return useMutation({
    mutationFn: animationGenerationApi.generate,
  });
}
