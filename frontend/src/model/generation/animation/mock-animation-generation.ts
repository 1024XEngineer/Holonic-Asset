import { DataApiError } from "@/lib/data-api-error";
import { runMockRequest } from "@/lib/mock-request";
import type { CharacterSpriteSheet } from "@/model/asset";

import type {
  GenerateAnimationInput,
  GenerateAnimationResult,
  SpriteAssetKind,
} from "./types";

type MockAnimationFixture = {
  assetKind: SpriteAssetKind;
  frameCount: number;
  spriteSheet: CharacterSpriteSheet;
};

const mockAnimationFixtures = new Map<string, MockAnimationFixture>([
  [
    "swordsman",
    {
      assetKind: "character",
      frameCount: 8,
      spriteSheet: {
        format: "png-sprite-sheet",
        imageUrl: "/assets/characters/swordsman/attack-front.png",
        frameWidth: 64,
        frameHeight: 64,
        columns: 8,
        rows: 1,
      },
    },
  ],
  [
    "knight",
    {
      assetKind: "character",
      frameCount: 5,
      spriteSheet: {
        format: "png-sprite-sheet",
        imageUrl: "/assets/characters/knight/attack-1.png",
        frameWidth: 128,
        frameHeight: 128,
        columns: 5,
        rows: 1,
      },
    },
  ],
  [
    "alchemy-table",
    {
      assetKind: "object",
      frameCount: 8,
      spriteSheet: {
        format: "png-sprite-sheet",
        imageUrl: "/assets/object/Alchemy_Table_02-Sheet.png",
        frameWidth: 48,
        frameHeight: 64,
        columns: 8,
        rows: 1,
      },
    },
  ],
]);

export function generateMockAnimation(
  input: GenerateAnimationInput,
): Promise<GenerateAnimationResult> {
  return runMockRequest(
    () => {
      const fixture = getMockAnimationFixture(input);

      return {
        generationId: `animation-${crypto.randomUUID()}`,
        animation: {
          kind: "clip",
          label: input.label.trim(),
          frameCount: fixture.frameCount,
          spriteSheet: structuredClone(fixture.spriteSheet),
        },
      };
    },
    { delayMs: 20_000 },
  );
}

function getMockAnimationFixture(input: GenerateAnimationInput) {
  const sourceAssetId = input.assetId.split("-copy-", 1)[0] ?? input.assetId;
  const fixture = mockAnimationFixtures.get(sourceAssetId);

  if (!fixture || fixture.assetKind !== input.assetKind) {
    throw new DataApiError(
      "NOT_FOUND",
      "Mock animation fixture was not found for this asset.",
      { assetId: input.assetId, assetKind: input.assetKind },
    );
  }

  return fixture;
}
