export type QuickGenerationDraft<Reference = unknown> = {
  assetId?: string;
  prompt: string;
  size: string;
  reference?: Reference;
  referenceFileName?: string;
};
