import { queryOptions, useQuery } from "@tanstack/react-query";

import { assetWorkspaceApi } from "./record.api";
import { recordKeys } from "./record.keys";
import { readAuthenticatedUserId } from "@/model/auth";

export function recordQueryOptions(projectId: string, assetId: string) {
  const userID = readAuthenticatedUserId();
  return queryOptions({
    queryKey: recordKeys.detail(userID, projectId, assetId),
    queryFn: () => assetWorkspaceApi.load({ projectId, assetId }),
  });
}

export function useRecordQuery(projectId: string, assetId: string) {
  return useQuery(recordQueryOptions(projectId, assetId));
}
