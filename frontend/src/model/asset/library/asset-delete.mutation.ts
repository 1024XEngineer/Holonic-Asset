import { useMutation, useQueryClient } from "@tanstack/react-query";

import { assetApi } from "./asset.api";
import { assetKeys } from "./keys";

type DeleteAssetInput = { projectId: string; assetId: string };

export function useDeleteAssetMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ projectId, assetId }: DeleteAssetInput) =>
      assetApi.delete(projectId, assetId),
    onSuccess: (assetGroups, { projectId }) => {
      queryClient.setQueryData(assetKeys.library(projectId), assetGroups);
    },
  });
}
