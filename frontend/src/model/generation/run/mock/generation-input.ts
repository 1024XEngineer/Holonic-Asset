import type { CreationRequest } from "@/features/generation/types";

export type GenerationInput = {
  projectId: string;
  request: CreationRequest;
};
