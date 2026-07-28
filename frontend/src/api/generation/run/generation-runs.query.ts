import { useQuery } from "@tanstack/react-query";

import { generationApi } from "./generation.api";
import { generationKeys } from "./keys";

export function useGenerationRunsQuery(projectId: string | undefined) {
  return useQuery({
    queryKey: generationKeys.runs(projectId ?? "unselected"),
    queryFn: () => generationApi.listRuns(projectId!),
    enabled: Boolean(projectId),
  });
}
