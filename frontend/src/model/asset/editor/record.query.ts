import { queryOptions, useQuery } from "@tanstack/react-query";

import { recordApi } from "./record.api";
import { recordKeys } from "./record.keys";

export function recordQueryOptions(projectId: string, assetId: string) {
  return queryOptions({
    queryKey: recordKeys.detail(projectId, assetId),
    queryFn: () => recordApi.get({ projectId, assetId }),
  });
}

export function useRecordQuery(projectId: string, assetId: string) {
  return useQuery(recordQueryOptions(projectId, assetId));
}
