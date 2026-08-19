import { useQuery } from "@tanstack/react-query";

import { readAuthenticatedUserId } from "@/model/auth";
import { coreGenerationApi } from "./core-generation.api";
import { generationKeys } from "./keys";

export function useGenerationCandidateQuery<Content = unknown>(
  runId: string | undefined,
) {
  const userId = readAuthenticatedUserId();
  return useQuery({
    queryKey: generationKeys.candidate(userId, runId ?? "unselected"),
    queryFn: () => coreGenerationApi.detail<Content>(positiveRunId(runId!)),
    enabled: Boolean(runId),
    staleTime: Infinity,
  });
}

function positiveRunId(value: string) {
  const id = Number(value);
  if (!Number.isSafeInteger(id) || id <= 0) {
    throw new Error("Generation candidate requires a persisted run.");
  }
  return id;
}
