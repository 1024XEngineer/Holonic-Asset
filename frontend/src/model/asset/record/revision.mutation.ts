import { useMutation, useQueryClient } from "@tanstack/react-query";

import { assetKeys } from "../library/keys";
import type { AssetWorkspaceData } from "./types";
import { assetRecordApi } from "./record.api";
import { recordKeys } from "./record.keys";
import { readAuthenticatedUserId } from "@/model/auth";

export function useSaveAssetRevisionMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: assetRecordApi.saveRevision,
    onSuccess: async (saved, { assetId, projectId }) => {
      const userID = readAuthenticatedUserId();
      queryClient.setQueryData(
        recordKeys.detail(userID, projectId, assetId),
        (current: AssetWorkspaceData | undefined) =>
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
          queryKey: assetKeys.library(userID, projectId),
        }),
        queryClient.invalidateQueries({
          queryKey: recordKeys.detail(userID, projectId, assetId),
        }),
      ]);
    },
  });
}
