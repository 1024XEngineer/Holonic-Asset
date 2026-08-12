import { useQuery, useQueryClient } from "@tanstack/react-query";

import { projectApi } from "./project.api";
import { projectKeys } from "./keys";
import type { ProjectSummary } from "./types";

export function useProjectDetailQuery(projectId: string | undefined) {
  const queryClient = useQueryClient();
  const listQueryKey = projectKeys.list();

  return useQuery({
    queryKey: projectKeys.detail(projectId ?? "unselected"),
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
