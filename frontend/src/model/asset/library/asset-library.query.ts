import { useQuery } from "@tanstack/react-query";

import { assetApi } from "./asset.api";
import { assetKeys } from "./keys";
import { readAuthenticatedUserId } from "@/model/auth";

export function useAssetLibraryQuery(projectId: string | undefined) {
  const userID = readAuthenticatedUserId();
  return useQuery({
    queryKey: assetKeys.library(userID, projectId ?? "unselected"),
    queryFn: () => assetApi.listGroups(projectId!),
    enabled: Boolean(projectId),
  });
}
