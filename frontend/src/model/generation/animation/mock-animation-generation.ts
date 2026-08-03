import type { GenerateAnimationInput, GenerateAnimationResult } from "./types";
import { runMockRequest } from "@/lib/mock-request";

export function generateMockAnimation(
  input: GenerateAnimationInput,
): Promise<GenerateAnimationResult> {
  return runMockRequest(
    () => {
      const isLargeCharacter = input.prototype.frameWidth >= 128;
      const frameCount = isLargeCharacter ? 5 : 8;

      return {
        generationId: `animation-${crypto.randomUUID()}`,
        animation: {
          kind: "clip",
          label: input.label.trim(),
          frameCount,
          spriteSheet: {
            ...input.prototype,
            imageUrl: isLargeCharacter
              ? "/assets/characters/knight/attack-1.png"
              : "/assets/characters/swordsman/attack-front.png",
            columns: frameCount,
            rows: 1,
            row: undefined,
          },
        },
      };
    },
    { delayMs: 20_000 },
  );
}
