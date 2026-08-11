import type {
  GenerateQuickAssetInput,
  QuickGenerationAsset,
} from "@/model/generation";
import { generateQuickAssetInputSchema } from "@/model/generation";

export const quickGenerationSizes = [
  "32 × 32 px",
  "64 × 64 px",
  "128 × 128 px",
  "256 × 256 px",
  "512 × 512 px",
] as const;

export const defaultQuickGenerationSize = quickGenerationSizes[1];

export type QuickGenerationDraft<Reference = unknown> = {
  assetId?: string;
  prompt: string;
  size: string;
  reference?: Reference;
  referenceFileName?: string;
};

export function createQuickGenerationDraft<Reference = unknown>(
  asset?: QuickGenerationAsset,
  reference?: Reference,
): QuickGenerationDraft<Reference> {
  return {
    assetId: asset?.id,
    prompt: "",
    size: asset?.size ?? defaultQuickGenerationSize,
    reference,
    referenceFileName: asset?.referenceFileName,
  };
}

export function toGenerateQuickAssetInput(
  draft: QuickGenerationDraft,
): GenerateQuickAssetInput | undefined {
  const result = generateQuickAssetInputSchema.safeParse({
    assetId: draft.assetId,
    prompt: draft.prompt,
    size: draft.size,
    referenceFileName: draft.referenceFileName,
  });

  return result.success ? result.data : undefined;
}
