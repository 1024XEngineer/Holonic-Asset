import { z } from "zod";

export type QuickGenerationAsset = {
  id: string;
  prompt: string;
  size: string;
  referenceFileName?: string;
  previewUrl?: string;
};

export const generateQuickAssetInputSchema = z.object({
  assetId: z.string().optional(),
  prompt: z.string().trim().min(1, "A quick generation prompt is required."),
  size: z.string().trim().min(1, "A quick generation size is required."),
  referenceFileName: z.string().optional(),
});

export type GenerateQuickAssetInput = z.infer<
  typeof generateQuickAssetInputSchema
>;
