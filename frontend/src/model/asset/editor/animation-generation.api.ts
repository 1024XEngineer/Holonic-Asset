import type { GenerateAnimationInput, GenerateAnimationResult } from "./types";

import { generateMockAnimation } from "./mock/animation-generation";

export type AnimationGenerationApi = {
  generate: (input: GenerateAnimationInput) => Promise<GenerateAnimationResult>;
};

export const animationGenerationApi: AnimationGenerationApi = {
  generate: generateMockAnimation,
};
