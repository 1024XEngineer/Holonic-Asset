import { queryOptions, useQuery } from "@tanstack/react-query";

import { quickGenerationApi } from "./quick-generation.api";
import { quickGenerationKeys } from "./quick-generation.keys";

export function quickAssetsQueryOptions() {
  return queryOptions({
    queryKey: quickGenerationKeys.assets(),
    queryFn: quickGenerationApi.listAssets,
  });
}

export function useQuickAssetsQuery() {
  return useQuery(quickAssetsQueryOptions());
}
