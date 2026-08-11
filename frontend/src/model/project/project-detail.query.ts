import { useQuery } from "@tanstack/react-query";

import { projectApi } from "./project.api";
import { projectKeys } from "./keys";

export function useProjectDetailQuery(projectId: string | undefined) {
  return useQuery({
    queryKey: projectKeys.detail(projectId ?? "unselected"),
    queryFn: () => projectApi.detail(projectId!),
    enabled: Boolean(projectId),
  });
}
