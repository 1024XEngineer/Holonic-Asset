import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import type { GenerateAnimationInput } from "./types";
import { generateMockAnimation } from "./mock-animation-generation";

describe("generateMockAnimation", () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => vi.useRealTimers());

  it.each([
    ["swordsman", 128, "/assets/characters/swordsman/attack-front.png", 8, 64],
    ["knight", 64, "/assets/characters/knight/attack-1.png", 5, 128],
  ])(
    "selects the %s character fixture by asset id",
    async (assetId, prototypeWidth, imageUrl, frameCount, frameWidth) => {
      const result = await generate(
        animationInput({ assetId, prototypeWidth, assetKind: "character" }),
      );

      expect(result.animation).toMatchObject({
        frameCount,
        spriteSheet: { imageUrl, frameWidth },
      });
    },
  );

  it("resolves copied assets to their source fixture", async () => {
    const result = await generate(
      animationInput({
        assetId: "alchemy-table-copy-example",
        assetKind: "object",
        prototypeWidth: 64,
      }),
    );

    expect(result.animation).toMatchObject({
      frameCount: 8,
      spriteSheet: {
        imageUrl: "/assets/object/Alchemy_Table_02-Sheet.png",
        frameWidth: 48,
        frameHeight: 64,
      },
    });
  });

  it("rejects assets without a matching fixture", async () => {
    const result = generateMockAnimation(
      animationInput({ assetId: "unknown-object", assetKind: "object" }),
    );
    const assertion = expect(result).rejects.toMatchObject({
      code: "NOT_FOUND",
      details: { assetId: "unknown-object", assetKind: "object" },
    });

    await vi.advanceTimersByTimeAsync(20_000);
    await assertion;
  });
});

function animationInput({
  assetId,
  assetKind,
  prototypeWidth = 64,
}: {
  assetId: string;
  assetKind: GenerateAnimationInput["assetKind"];
  prototypeWidth?: number;
}): GenerateAnimationInput {
  return {
    projectId: "moonlit-orchard",
    assetId,
    assetKind,
    label: "Attack",
    prompt: "A fast attack",
    prototype: {
      format: "png-sprite-sheet",
      imageUrl: "/prototype.png",
      frameWidth: prototypeWidth,
      frameHeight: prototypeWidth,
      columns: 1,
      rows: 1,
    },
  };
}

async function generate(input: GenerateAnimationInput) {
  const result = generateMockAnimation(input);
  await vi.advanceTimersByTimeAsync(20_000);
  return result;
}
