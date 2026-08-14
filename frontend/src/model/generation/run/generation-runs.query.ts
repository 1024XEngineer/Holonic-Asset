import { useEffect, useRef } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";

import { refreshAssetLibraryCache } from "../../asset/library/asset-library-cache";
import { recordQueryOptions } from "../../asset/record/record.query";
import { projectKeys } from "../../project/keys";
import { coreGenerationApi } from "./core-generation.api";
import { forgetGenerationRunMetadata, generationApi } from "./generation.api";
import { generationKeys } from "./keys";
import {
  findSettledGenerationRunIds,
  generationPollingInterval,
} from "./generation-polling";
import type { GenerationRun } from "./types";
import { readAuthenticatedUserId } from "@/model/auth";

const MAX_RECONCILIATION_ATTEMPTS = 3;

export function useGenerationRunsQuery(
  projectId: string | undefined,
  assetId?: string,
) {
  const userID = readAuthenticatedUserId();
  const queryClient = useQueryClient();
  const reconcilingRunIds = useRef(new Set<string>());
  const handledRunIds = useRef(new Set<string>());
  const reconciliationAttempts = useRef(new Map<string, number>());
  const previousRuns = useRef<{
    projectId: string | undefined;
    assetId: string | undefined;
    runs: GenerationRun[];
  }>({ projectId, assetId, runs: [] });
  const query = useQuery({
    queryKey: generationKeys.runs(userID, projectId ?? "unselected", assetId),
    queryFn: () => generationApi.listRuns(projectId!, assetId),
    enabled: Boolean(projectId),
    refetchInterval: ({ state }) => generationPollingInterval(state.data),
  });

  useEffect(() => {
    if (
      previousRuns.current.projectId !== projectId ||
      previousRuns.current.assetId !== assetId
    ) {
      previousRuns.current = { projectId, assetId, runs: query.data ?? [] };
      reconcilingRunIds.current.clear();
      handledRunIds.current.clear();
      reconciliationAttempts.current.clear();
      return;
    }
    if (!projectId || !query.data) return;

    const settledRunIds = findSettledGenerationRunIds(
      previousRuns.current.runs,
      query.data,
    ).filter(
      (runId) =>
        !reconcilingRunIds.current.has(runId) &&
        !handledRunIds.current.has(runId),
    );
    const previousById = new Map(
      previousRuns.current.runs.map((run) => [run.id, run]),
    );
    const settledRuns = settledRunIds.flatMap((runId) => {
      const run = previousById.get(runId);
      return run ? [run] : [];
    });
    const runsToReconcile = new Map(settledRuns.map((run) => [run.id, run]));
    for (const run of query.data) {
      if (
        run.status === "failed" &&
        !run.error &&
        !reconcilingRunIds.current.has(run.id) &&
        !handledRunIds.current.has(run.id)
      ) {
        runsToReconcile.set(run.id, run);
      }
    }
    previousRuns.current = { projectId, assetId, runs: query.data };

    const reconciliationRuns = [...runsToReconcile.values()];
    if (reconciliationRuns.length === 0) return;

    const queryKey = generationKeys.runs(userID, projectId, assetId);
    for (const run of reconciliationRuns) {
      reconcilingRunIds.current.add(run.id);
    }
    queryClient.setQueryData<GenerationRun[]>(queryKey, (current = []) => [
      ...current,
      ...reconciliationRuns.filter(
        (run) => !current.some((currentRun) => currentRun.id === run.id),
      ),
    ]);

    void Promise.all(
      reconciliationRuns.map(async (run) => {
        const attempt = (reconciliationAttempts.current.get(run.id) ?? 0) + 1;
        reconciliationAttempts.current.set(run.id, attempt);
        try {
          const detail = await coreGenerationApi.detail(coreRunId(run.id));
          if (detail.status === "completed") {
            handledRunIds.current.add(run.id);
            reconciliationAttempts.current.delete(run.id);
            forgetGenerationRunMetadata(projectId, [run.id]);
            removeRun(queryClient, queryKey, run.id);
            await refreshSettledAssets(queryClient, userID, projectId, assetId);
            return;
          }
          if (detail.status === "failed") {
            handledRunIds.current.add(run.id);
            reconciliationAttempts.current.delete(run.id);
            queryClient.setQueryData<GenerationRun[]>(
              queryKey,
              (current = []) =>
                current.map((currentRun) =>
                  currentRun.id === run.id
                    ? { ...currentRun, status: "failed", error: detail.error }
                    : currentRun,
                ),
            );
            return;
          }
          if (detail.status === "cancelled") {
            forgetGenerationRunMetadata(projectId, [run.id]);
            handledRunIds.current.add(run.id);
            reconciliationAttempts.current.delete(run.id);
            removeRun(queryClient, queryKey, run.id);
            return;
          }
          if (attempt >= MAX_RECONCILIATION_ATTEMPTS) {
            reconciliationAttempts.current.delete(run.id);
            removeRun(queryClient, queryKey, run.id);
          }
        } catch {
          if (handledRunIds.current.has(run.id)) return;
          if (attempt >= MAX_RECONCILIATION_ATTEMPTS) {
            reconciliationAttempts.current.delete(run.id);
            removeRun(queryClient, queryKey, run.id);
            return;
          }
          queryClient.setQueryData<GenerationRun[]>(queryKey, (current = []) =>
            current.some((currentRun) => currentRun.id === run.id)
              ? current
              : [...current, run],
          );
        } finally {
          reconcilingRunIds.current.delete(run.id);
        }
      }),
    );
  }, [assetId, projectId, query.data, query.dataUpdatedAt, queryClient]);

  return query;
}

async function refreshSettledAssets(
  queryClient: ReturnType<typeof useQueryClient>,
  userID: number,
  projectId: string,
  assetId: string | undefined,
) {
  const refreshes: Promise<unknown>[] = [
    refreshAssetLibraryCache(queryClient, userID, projectId),
    queryClient.invalidateQueries({ queryKey: projectKeys.list(userID) }),
  ];
  if (assetId) {
    refreshes.push(
      queryClient.refetchQueries(
        {
          queryKey: recordQueryOptions(projectId, assetId).queryKey,
          type: "all",
        },
        { throwOnError: true },
      ),
    );
  }
  await Promise.all(refreshes);
}

function removeRun(
  queryClient: ReturnType<typeof useQueryClient>,
  queryKey: ReturnType<typeof generationKeys.runs>,
  runId: string,
) {
  queryClient.setQueryData<GenerationRun[]>(queryKey, (current = []) =>
    current.filter((run) => run.id !== runId),
  );
}

function coreRunId(runId: string) {
  const value = Number(runId);
  if (!Number.isSafeInteger(value) || value <= 0) {
    throw new Error("Generation status requires a persisted Core API run.");
  }
  return value;
}
