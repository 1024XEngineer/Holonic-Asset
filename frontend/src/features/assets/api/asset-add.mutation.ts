import { useMutation, useQueryClient } from "@tanstack/react-query";

import { assetApi } from "./asset.api";
import type { ProjectAsset } from "@/features/assets/domain";
import type { AssetKind } from "@/features/assets/domain";
import { assetKeys } from "./keys";

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
    onSuccess: (assetGroups, { projectId }) => {
      queryClient.setQueryData(assetKeys.library(projectId), assetGroups);
    },
  });
}
