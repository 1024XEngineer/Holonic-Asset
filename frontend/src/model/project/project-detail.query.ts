import { useQuery, useQueryClient } from "@tanstack/react-query";

import { projectApi } from "./project.api";
import { projectKeys } from "./keys";
import type { ProjectSummary } from "./types";
import { readAuthenticatedUserId } from "@/model/auth";

export function useProjectDetailQuery(projectId: string | undefined) {
  const queryClient = useQueryClient();
  const userID = readAuthenticatedUserId();
  const listQueryKey = projectKeys.list(userID);

  return useQuery({
    queryKey: projectKeys.detail(userID, projectId ?? "unselected"),
    queryFn: () => projectApi.detail(projectId!),
    enabled: Boolean(projectId),
    initialData: () =>
      queryClient
        .getQueryData<ProjectSummary[]>(listQueryKey)
        ?.find((project) => project.id === projectId),
    initialDataUpdatedAt: () =>
      queryClient.getQueryState(listQueryKey)?.dataUpdatedAt,
  });
}
