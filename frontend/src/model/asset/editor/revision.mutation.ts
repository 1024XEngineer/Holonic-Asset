import { useMutation, useQueryClient } from "@tanstack/react-query";

import { assetKeys } from "../library/keys";
import type { EditorWorkspaceData } from "@/features/asset-editor/types";
import { recordApi } from "./record.api";
import { recordKeys } from "./record.keys";

export function useSaveAssetRevisionMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: recordApi.saveRevision,
    onSuccess: async (saved, { assetId, projectId }) => {
      queryClient.setQueryData(
        recordKeys.detail(projectId, assetId),
        (current: EditorWorkspaceData | undefined) =>
          current
            ? {
                ...current,
                asset: {
                  ...current.asset,
                  version: saved.version,
                  history: structuredClone(saved.history),
                },
                record: structuredClone(saved.record),
              }
            : current,
      );
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: assetKeys.library(projectId),
        }),
        queryClient.invalidateQueries({
          queryKey: recordKeys.detail(projectId, assetId),
        }),
      ]);
    },
  });
}
