import type { QueryClient, QueryKey } from "@tanstack/react-query";

import { generationKeys } from "./keys";
import type { GenerationRun } from "./types";

type RemovedGenerationRun = {
  queryKey: QueryKey;
  run: GenerationRun;
  index: number;
};

export type GenerationRunCacheSnapshot = {
  removed: RemovedGenerationRun[];
};

export function removeGenerationRunFromCache(
  queryClient: QueryClient,
  userId: number,
  projectId: string,
  runId: string,
): GenerationRunCacheSnapshot {
  const removed: RemovedGenerationRun[] = [];
  const queryKey = generationKeys.runs(userId, projectId);

  for (const [cachedQueryKey, runs] of queryClient.getQueriesData<
    GenerationRun[]
  >({
    queryKey,
  })) {
    if (!runs) continue;
    const index = runs.findIndex((run) => run.id === runId);
    if (index < 0) continue;

    removed.push({ queryKey: cachedQueryKey, run: runs[index], index });
    queryClient.setQueryData<GenerationRun[]>(cachedQueryKey, (current = []) =>
      current.filter((run) => run.id !== runId),
    );
  }

  return { removed };
}

export function restoreGenerationRunCache(
  queryClient: QueryClient,
  snapshot: GenerationRunCacheSnapshot,
) {
  for (const { queryKey, run, index } of snapshot.removed) {
    queryClient.setQueryData<GenerationRun[]>(queryKey, (current) => {
      if (!current || current.some((currentRun) => currentRun.id === run.id)) {
        return current;
      }
      const runs = [...current];
      runs.splice(index, 0, run);
      return runs;
    });
  }
}

export function refreshGenerationRunCacheInBackground(
  queryClient: QueryClient,
  userId: number,
  projectId: string,
) {
  void (async () => {
    try {
      await queryClient.refetchQueries({
        queryKey: generationKeys.runs(userId, projectId),
        type: "all",
      });
    } catch {
      // Background refresh is best effort.
    }
  })();
}
