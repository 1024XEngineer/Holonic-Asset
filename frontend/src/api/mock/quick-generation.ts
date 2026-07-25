import { DataApiError } from "@/api/api-error";
import { runMockRequest, type MockRequestOptions } from "./request";
import { quickGenerationAssetSeed } from "./quick-generation.seed";
import type {
  GenerateQuickAssetInput,
  QuickGenerationAsset,
} from "@/domain/quick-generation";

const GENERATION_DELAY_MS = 700;

let assets = createSeedAssets();

export function listMockQuickAssets(options?: MockRequestOptions) {
  return runMockRequest(() => structuredClone(assets), options);
}

export function generateMockQuickAsset(
  input: GenerateQuickAssetInput,
  options?: MockRequestOptions,
) {
  return runMockRequest(
    () => {
      const prompt = input.prompt.trim();
      const size = input.size.trim();
      if (!prompt) {
        throw new DataApiError(
          "BAD_REQUEST",
          "A quick generation prompt is required.",
        );
      }
      if (!size) {
        throw new DataApiError(
          "BAD_REQUEST",
          "A quick generation size is required.",
        );
      }

      if (input.assetId) {
        const current = findAsset(input.assetId);
        const updated: QuickGenerationAsset = {
          ...current,
          prompt,
          size,
          referenceFileName: input.referenceFileName,
        };
        assets = assets.map((asset) =>
          asset.id === updated.id ? updated : asset,
        );
        return structuredClone(updated);
      }

      const created: QuickGenerationAsset = {
        id: `quick-asset-${crypto.randomUUID()}`,
        prompt,
        size,
        referenceFileName: input.referenceFileName,
      };
      assets = [...assets, created];
      return structuredClone(created);
    },
    { delayMs: options?.delayMs ?? GENERATION_DELAY_MS },
  );
}

export function deleteMockQuickAsset(
  assetId: string,
  options?: MockRequestOptions,
) {
  return runMockRequest(() => {
    findAsset(assetId);
    assets = assets.filter((asset) => asset.id !== assetId);
  }, options);
}

export function resetQuickGenerationMockData() {
  assets = createSeedAssets();
}

function createSeedAssets(): QuickGenerationAsset[] {
  return quickGenerationAssetSeed.map((asset) => ({ ...asset }));
}

function findAsset(assetId: string) {
  const asset = assets.find((item) => item.id === assetId);
  if (!asset) {
    throw new DataApiError(
      "NOT_FOUND",
      "Quick generation asset was not found.",
      { assetId },
    );
  }
  return asset;
}
