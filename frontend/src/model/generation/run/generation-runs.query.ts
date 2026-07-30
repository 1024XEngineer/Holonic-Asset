import { useEffect, useRef } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";

import { assetKeys } from "../../asset/library/keys";
import { projectKeys } from "../../project/keys";
import { generationApi } from "./generation.api";
import { generationKeys } from "./keys";
import {
  findSettledGenerationRunIds,
  generationPollingInterval,
} from "./generation-polling";
import type { GenerationRun } from "@/features/generation/types";

export function useGenerationRunsQuery(projectId: string | undefined) {
  const queryClient = useQueryClient();
  const previousRuns = useRef<{
    projectId: string | undefined;
    runs: GenerationRun[];
  }>({ projectId, runs: [] });
  const query = useQuery({
    queryKey: generationKeys.runs(projectId ?? "unselected"),
    queryFn: () => generationApi.listRuns(projectId!),
    enabled: Boolean(projectId),
    refetchInterval: ({ state }) => generationPollingInterval(state.data),
  });

  useEffect(() => {
    if (previousRuns.current.projectId !== projectId) {
      previousRuns.current = { projectId, runs: query.data ?? [] };
      return;
    }
    if (!projectId || !query.data) return;

    const settledRunIds = findSettledGenerationRunIds(
      previousRuns.current.runs,
      query.data,
    );
    previousRuns.current = { projectId, runs: query.data };

    if (settledRunIds.length === 0) return;

    void Promise.all([
      queryClient.invalidateQueries({
        queryKey: assetKeys.library(projectId),
      }),
      queryClient.invalidateQueries({ queryKey: projectKeys.list() }),
    ]);
  }, [projectId, query.data, query.dataUpdatedAt, queryClient]);

  return query;
}
