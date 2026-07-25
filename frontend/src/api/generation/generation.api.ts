import { listMockGenerationRuns, mockGenerationLifecycle } from "@/api/mock";
import type {
  GenerationInput,
  GenerationLifecycleUpdate,
} from "./generation-lifecycle";

export const generationApi = {
  listRuns: (projectId: string) => listMockGenerationRuns(projectId),
  enqueue: (
    input: GenerationInput,
    onUpdate: (update: GenerationLifecycleUpdate) => void,
  ) => mockGenerationLifecycle.enqueue(input, onUpdate),
};
