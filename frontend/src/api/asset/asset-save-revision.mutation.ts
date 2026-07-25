import { useMutation, useQueryClient } from "@tanstack/react-query";

import { assetApi } from "./asset.api";
import { recordKeys } from "@/api/record/record.keys";
import type { RecordContent, RecordData } from "@/domain/asset";
import { assetKeys } from "./keys";

type SaveAssetRevisionInput = {
  projectId: string;
  assetId: string;
  content: RecordContent;
};

export function useSaveAssetRevisionMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ projectId, assetId, content }: SaveAssetRevisionInput) =>
      assetApi.saveRevision(projectId, assetId, content),
    onSuccess: async (assetGroups, { assetId, content, projectId }) => {
      queryClient.setQueryData(assetKeys.library(projectId), assetGroups);
      const savedAsset = assetGroups
        .flatMap((group) => group.assets)
        .find((asset) => asset.id === assetId);
      queryClient.setQueryData(
        recordKeys.detail(projectId, assetId),
        (current: RecordData | undefined) =>
          current && savedAsset
            ? {
                ...current,
                asset: {
                  ...current.asset,
                  version: savedAsset.version,
                  history: structuredClone(savedAsset.history),
                },
                content: structuredClone(content),
              }
            : current,
      );
      await queryClient.invalidateQueries({
        queryKey: recordKeys.detail(projectId, assetId),
      });
    },
  });
}
