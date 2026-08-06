import { useMutation, useQueryClient } from "@tanstack/react-query";

import type { AssetMetadataUpdate } from "../types";
import { assetApi } from "./asset.api";
import { createAssetLibraryCacheSync } from "./asset-library-cache";

type UpdateAssetInput = {
  projectId: string;
  assetId: string;
  metadata: AssetMetadataUpdate;
};

export function useUpdateAssetMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ projectId, assetId, metadata }: UpdateAssetInput) =>
      assetApi.update(projectId, assetId, metadata),
    onSuccess: createAssetLibraryCacheSync(queryClient),
  });
}
