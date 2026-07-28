import {
  mutationOptions,
  useMutation,
  useQueryClient,
  type QueryClient,
} from "@tanstack/react-query";

import type { QuickGenerationAsset } from "@/model";
import { quickGenerationApi } from "./quick-generation.api";
import { quickGenerationKeys } from "./quick-generation.keys";

export function generateQuickAssetMutationOptions(queryClient: QueryClient) {
  return mutationOptions({
    mutationFn: quickGenerationApi.generateAsset,
    onSuccess: (asset) => {
      queryClient.setQueryData<QuickGenerationAsset[]>(
        quickGenerationKeys.assets(),
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
      queryClient.setQueryData<QuickGenerationAsset[]>(
        quickGenerationKeys.assets(),
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
