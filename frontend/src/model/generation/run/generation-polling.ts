import type { GenerationRun } from "@/features/generation/types";

export const GENERATION_POLL_INTERVAL_MS = 1_000;

export function isGenerationRunActive(run: GenerationRun) {
  return run.status === "pending" || run.status === "processing";
}

export function generationPollingInterval(runs: GenerationRun[] | undefined) {
  return runs?.some(isGenerationRunActive)
    ? GENERATION_POLL_INTERVAL_MS
    : false;
}

export function findSettledGenerationRunIds(
  previous: GenerationRun[],
  current: GenerationRun[],
) {
  const currentActiveIds = new Set(
    current.filter(isGenerationRunActive).map((run) => run.id),
  );

  return previous
    .filter(isGenerationRunActive)
    .map((run) => run.id)
    .filter((runId) => !currentActiveIds.has(runId));
}
