import {
  deleteMockQuickAsset,
  generateMockQuickAsset,
  listMockQuickAssets,
} from "./mock/quick-generation";
import type { GenerateQuickAssetInput, QuickGenerationAsset } from "@/model";

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
