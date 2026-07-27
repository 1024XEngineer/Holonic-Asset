export type QuickGenerationAsset = {
  id: string;
  prompt: string;
  size: string;
  referenceFileName?: string;
  previewUrl?: string;
};

export type GenerateQuickAssetInput = {
  assetId?: string;
  prompt: string;
  size: string;
  referenceFileName?: string;
};
