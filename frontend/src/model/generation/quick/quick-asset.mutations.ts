import {
  mutationOptions,
  useMutation,
  useQueryClient,
  type QueryClient,
} from "@tanstack/react-query";

import type { QuickGenerationAsset } from "./types";
import { quickGenerationApi } from "./quick-generation.api";
import { quickGenerationKeys } from "./quick-generation.keys";
import { readAuthenticatedUserId } from "@/model/auth";

export function generateQuickAssetMutationOptions(queryClient: QueryClient) {
  return mutationOptions({
    mutationFn: quickGenerationApi.generateAsset,
    onSuccess: (asset) => {
      const userID = readAuthenticatedUserId();
      queryClient.setQueryData<QuickGenerationAsset[]>(
        quickGenerationKeys.assets(userID),
        (current = []) => {
          const exists = current.some((item) => item.id === asset.id);
          return exists
            ? current.map((item) => (item.id === asset.id ? asset : item))
            : [...current, asset];
        },
      );
    },
  });
}

export function deleteQuickAssetMutationOptions(queryClient: QueryClient) {
  return mutationOptions({
    mutationFn: quickGenerationApi.deleteAsset,
    onSuccess: (_, assetId) => {
      const userID = readAuthenticatedUserId();
      queryClient.setQueryData<QuickGenerationAsset[]>(
        quickGenerationKeys.assets(userID),
        (current = []) => current.filter((asset) => asset.id !== assetId),
      );
    },
  });
}

export function useGenerateQuickAssetMutation() {
  const queryClient = useQueryClient();
  return useMutation(generateQuickAssetMutationOptions(queryClient));
}

export function useDeleteQuickAssetMutation() {
  const queryClient = useQueryClient();
  return useMutation(deleteQuickAssetMutationOptions(queryClient));
}
