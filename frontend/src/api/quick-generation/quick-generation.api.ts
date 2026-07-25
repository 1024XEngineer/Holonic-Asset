import {
  deleteMockQuickAsset,
  generateMockQuickAsset,
  listMockQuickAssets,
} from "@/api/mock";
import type {
  GenerateQuickAssetInput,
  QuickGenerationAsset,
} from "@/domain/quick-generation";

export type QuickGenerationApi = {
  listAssets: () => Promise<QuickGenerationAsset[]>;
  generateAsset: (
    input: GenerateQuickAssetInput,
  ) => Promise<QuickGenerationAsset>;
  deleteAsset: (assetId: string) => Promise<void>;
};

export const quickGenerationApi: QuickGenerationApi = {
  listAssets: listMockQuickAssets,
  generateAsset: generateMockQuickAsset,
  deleteAsset: deleteMockQuickAsset,
};
