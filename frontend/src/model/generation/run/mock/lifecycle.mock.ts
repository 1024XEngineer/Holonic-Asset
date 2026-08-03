import { completeMockGeneration } from "./generation.mock";
import {
  createMockGenerationRun,
  removeMockGenerationRun,
  updateMockGenerationRun,
} from "./generation-runs.mock";
import { addMockAsset } from "../../../asset/library/mock";
import { hasMockProject } from "../../../project/mock";
import type { GenerationInput } from "./generation-input";

const MOCK_QUEUE_DELAY_MS = 650;

/** Starts background work and returns immediately, matching a pending HTTP job. */
export async function enqueueMockGeneration({
  projectId,
  request,
}: GenerationInput) {
  const pendingRun = createMockGenerationRun({ ...request, projectId });

  globalThis.setTimeout(() => {
    if (!hasMockProject(projectId)) {
      removeMockGenerationRun(pendingRun.id);
      return;
    }

    const processingRun = updateMockGenerationRun({
      ...pendingRun,
      status: "processing",
    });

    void completeMockGeneration(processingRun)
      .then(async ({ asset, kind }) => {
        if (hasMockProject(projectId)) {
          await addMockAsset(projectId, kind, asset);
        }
        removeMockGenerationRun(processingRun.id);
      })
      .catch(() => {
        updateMockGenerationRun({ ...processingRun, status: "failed" });
      });
  }, MOCK_QUEUE_DELAY_MS);

  return structuredClone(pendingRun);
}
