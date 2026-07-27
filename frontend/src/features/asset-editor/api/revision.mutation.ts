import { useMutation, useQueryClient } from "@tanstack/react-query";

import { saveMockAssetRevision } from "@/features/assets/api/mock";
import { assetKeys } from "@/features/assets/api";
import type { RecordContent, RecordData } from "@/features/assets/domain";
import { recordKeys } from "./record.keys";

type SaveAssetRevisionInput = {
  projectId: string;
  assetId: string;
  content: RecordContent;
};

export function useSaveAssetRevisionMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ projectId, assetId, content }: SaveAssetRevisionInput) =>
      saveMockAssetRevision(projectId, assetId, content),
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
