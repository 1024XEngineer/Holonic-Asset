import { useQuery } from "@tanstack/react-query";

import { assetApi } from "./asset.api";
import { assetKeys } from "./keys";

export function useAssetLibraryQuery(projectId: string | undefined) {
  return useQuery({
    queryKey: assetKeys.library(projectId ?? "unselected"),
    queryFn: () => assetApi.listGroups(projectId!),
    enabled: Boolean(projectId),
  });
}
