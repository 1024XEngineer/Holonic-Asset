import { queryOptions, useQuery } from "@tanstack/react-query";

import { quickGenerationApi } from "./quick-generation.api";
import { quickGenerationKeys } from "./quick-generation.keys";
import { readAuthenticatedUserId } from "@/model/auth";

export function quickAssetsQueryOptions() {
  const userID = readAuthenticatedUserId();
  return queryOptions({
    queryKey: quickGenerationKeys.assets(userID),
    queryFn: quickGenerationApi.listAssets,
  });
}

export function useQuickAssetsQuery() {
  return useQuery(quickAssetsQueryOptions());
}
