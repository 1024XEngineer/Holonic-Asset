export type QuickGenerationDraft<CreatingReference = unknown> = {
  assetId?: string;
  prompt: string;
  size: string;
  creatingReference?: CreatingReference;
  creatingReferenceFileName?: string;
};
