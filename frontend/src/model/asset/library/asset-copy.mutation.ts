import { useMutation, useQueryClient } from "@tanstack/react-query";

import { assetApi } from "./asset.api";
import { assetKeys } from "./keys";

type CopyAssetInput = { projectId: string; assetId: string };

export function useCopyAssetMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ projectId, assetId }: CopyAssetInput) =>
      assetApi.copy(projectId, assetId),
    onSuccess: (assetGroups, { projectId }) => {
      queryClient.setQueryData(assetKeys.library(projectId), assetGroups);
    },
  });
}
