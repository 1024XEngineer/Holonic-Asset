import { useMutation, useQueryClient } from "@tanstack/react-query";

import { readAuthenticatedUserId } from "@/model/auth";
import { assetKeys } from "../library/keys";
import { assetRecordApi } from "./record.api";
import { recordKeys } from "./record.keys";

export function useRollbackAssetRecordMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: assetRecordApi.rollback,
    onSuccess: async (_result, { assetId, projectId }) => {
      const userID = readAuthenticatedUserId();
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
