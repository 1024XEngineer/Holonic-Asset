import type { QuickGenerationAsset } from "@/types/quick-generation";

export const quickGenerationAssetSeed = [
  {
    id: "mushroom-courier",
    prompt:
      "A cheerful mushroom courier with a red cap, tiny satchel, and readable silhouette.",
    size: "64 × 64 px",
  },
  {
    id: "moonlit-lantern",
    prompt:
      "A small brass lantern glowing with cool blue moonlight and subtle engraved details.",
    size: "128 × 128 px",
  },
  {
    id: "forest-gate",
    prompt:
      "An ancient moss-covered stone gate framed by roots and tiny luminous flowers.",
    size: "256 × 256 px",
  },
] as const satisfies readonly QuickGenerationAsset[];
