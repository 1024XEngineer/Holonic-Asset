import { useMutation, useQueryClient } from "@tanstack/react-query";

import { assetApi } from "./asset.api";
import type { AssetKind, ProjectAsset } from "../types";
import { createAssetLibraryCacheSync } from "./asset-library-cache";

type AddAssetInput = {
  projectId: string;
  kind: AssetKind;
  asset: ProjectAsset;
};

export function useAddAssetMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ projectId, kind, asset }: AddAssetInput) =>
      assetApi.add(projectId, kind, asset),
    onSuccess: createAssetLibraryCacheSync(queryClient),
  });
}
