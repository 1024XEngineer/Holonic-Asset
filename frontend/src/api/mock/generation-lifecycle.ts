import { completeMockGeneration } from "./generation";
import {
  addMockAsset,
  createMockGenerationRun,
  hasMockProject,
  removeMockGenerationRun,
  updateMockGenerationRun,
} from "./workspace";
import { createGenerationLifecycle } from "@/api/generation/generation-lifecycle";

export const mockGenerationLifecycle = createGenerationLifecycle({
  createRun: createMockGenerationRun,
  updateRun: updateMockGenerationRun,
  removeRun: removeMockGenerationRun,
  completeGeneration: completeMockGeneration,
  hasProject: hasMockProject,
  addAsset: addMockAsset,
});
