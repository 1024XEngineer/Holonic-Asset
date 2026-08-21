import type {
  GenerateQuickAssetInput,
  QuickGenerationAsset,
} from "@/model/generation";
import { generateQuickAssetInputSchema } from "@/model/generation";

import type { QuickGenerationDraft } from "../types";

export const quickGenerationSizes = [
  "32 × 32 px",
  "64 × 64 px",
  "128 × 128 px",
  "256 × 256 px",
  "512 × 512 px",
] as const;

export const defaultQuickGenerationSize = quickGenerationSizes[1];

export function createQuickGenerationDraft<CreatingReference = unknown>(
  asset?: QuickGenerationAsset,
  creatingReference?: CreatingReference,
): QuickGenerationDraft<CreatingReference> {
  return {
    assetId: asset?.id,
    prompt: "",
    size: asset?.size ?? defaultQuickGenerationSize,
    creatingReference,
    creatingReferenceFileName: asset?.creatingReferenceFileName,
  };
}

export function toGenerateQuickAssetInput(
  draft: QuickGenerationDraft,
): GenerateQuickAssetInput | undefined {
  const result = generateQuickAssetInputSchema.safeParse({
    assetId: draft.assetId,
    prompt: draft.prompt,
    size: draft.size,
    creatingReferenceFileName: draft.creatingReferenceFileName,
  });

  return result.success ? result.data : undefined;
}
