import { useMutation, useQueryClient } from "@tanstack/react-query";

import { assetApi } from "./asset.api";
import { createAssetLibraryCacheSync } from "./asset-library-cache";

type CopyAssetInput = { projectId: string; assetId: string };

export function useCopyAssetMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ projectId, assetId }: CopyAssetInput) =>
      assetApi.copy(projectId, assetId),
    onSuccess: createAssetLibraryCacheSync(queryClient),
  });
}
