import { listMockGenerationRuns, mockGenerationLifecycle } from "./mock";
import type { GenerationInput, GenerationLifecycleUpdate } from "@/model";

export const generationApi = {
  listRuns: (projectId: string) => listMockGenerationRuns(projectId),
  enqueue: (
    input: GenerationInput,
    onUpdate: (update: GenerationLifecycleUpdate) => void,
  ) => mockGenerationLifecycle.enqueue(input, onUpdate),
};
