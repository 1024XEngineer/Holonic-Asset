import type { GenerationRun } from "../types";

let generationRuns: GenerationRun[] = [];

export async function listMockGenerationRuns(projectId: string) {
  return structuredClone(
    generationRuns.filter((run) => run.projectId === projectId),
  );
}

export function createMockGenerationRun(
  run: Omit<GenerationRun, "id" | "status">,
) {
  const createdRun: GenerationRun = {
    ...structuredClone(run),
    id: crypto.randomUUID(),
    status: "pending",
  };
  generationRuns = [...generationRuns, createdRun];
  return structuredClone(createdRun);
}

export function updateMockGenerationRun(run: GenerationRun) {
  generationRuns = generationRuns.map((item) =>
    item.id === run.id ? structuredClone(run) : item,
  );
  return structuredClone(run);
}

export function removeMockGenerationRun(runId: string) {
  generationRuns = generationRuns.filter((run) => run.id !== runId);
}

export function deleteMockProjectGenerationRuns(projectId: string) {
  generationRuns = generationRuns.filter((run) => run.projectId !== projectId);
}

export function resetMockGenerationRuns() {
  generationRuns = [];
}
