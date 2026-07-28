import { completeMockGeneration } from "./generation.mock";
import {
  createMockGenerationRun,
  removeMockGenerationRun,
  updateMockGenerationRun,
} from "./generation-runs.mock";
import { addMockAsset } from "../../../asset/library/mock";
import { createGenerationLifecycle } from "@/model";
import { hasMockProject } from "../../../project/mock";

export const mockGenerationLifecycle = createGenerationLifecycle({
  createRun: createMockGenerationRun,
  updateRun: updateMockGenerationRun,
  removeRun: removeMockGenerationRun,
  completeGeneration: completeMockGeneration,
  hasProject: hasMockProject,
  addAsset: addMockAsset,
});
