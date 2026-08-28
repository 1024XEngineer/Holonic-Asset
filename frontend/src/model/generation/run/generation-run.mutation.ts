import { useMutation, useQueryClient } from "@tanstack/react-query";

import { getDefaultAssetCanvasSize, type AssetKind } from "../../asset";
import {
  coreGenerationApi,
  generationApi,
  rememberGenerationRunMetadata,
  type CreateGenerationRequest,
} from "./generation.api";
import type { GenerationInput, GenerationRun } from "./types";
import { generationKeys } from "./keys";
import { readAuthenticatedUserId } from "@/model/auth";

type AssetEditGenerationInput = {
  projectId: string;
  assetId: string;
  assetKind: AssetKind;
  assetName: string;
  prompt: string;
  request: CreateGenerationRequest;
};

export function useEnqueueGenerationMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: GenerationInput) => generationApi.enqueue(input),
    onSuccess: (run) => {
      const userID = readAuthenticatedUserId();
      upsertGenerationRun(
        queryClient,
        generationKeys.runs(userID, run.projectId),
        run,
      );
    },
  });
}

export function useEnqueueAssetEditGenerationMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: enqueueAssetEditGeneration,
    onSuccess: (run) => {
      upsertGenerationRun(
        queryClient,
        generationKeys.runs(
          readAuthenticatedUserId(),
          run.projectId,
          run.assetId,
        ),
        run,
      );
    },
  });
}

async function enqueueAssetEditGeneration(
  input: AssetEditGenerationInput,
): Promise<GenerationRun> {
  const projectId = positiveCoreId(input.projectId, "project");
  positiveCoreId(input.assetId, "asset");
  const created = await coreGenerationApi.create(projectId, input.request);
  const name = `Edit ${input.assetName}`;
  rememberGenerationRunMetadata(input.projectId, created.generationRunId, {
    kind: input.assetKind,
    name,
    prompt: input.prompt,
    assetId: input.assetId,
  });
  return {
    id: String(created.generationRunId),
    projectId: input.projectId,
    assetId: input.assetId,
    kind: input.assetKind,
    name,
    prompt: input.prompt,
    canvasSize: getDefaultAssetCanvasSize(input.assetKind),
    status: "pending",
  };
}

function upsertGenerationRun(
  queryClient: ReturnType<typeof useQueryClient>,
  queryKey: ReturnType<typeof generationKeys.runs>,
  run: GenerationRun,
) {
  queryClient.setQueryData<GenerationRun[]>(queryKey, (current = []) => [
    ...current.filter((currentRun) => currentRun.id !== run.id),
    run,
  ]);
}

function positiveCoreId(value: string, resource: string) {
  const id = Number(value);
  if (!Number.isSafeInteger(id) || id <= 0) {
    throw new Error(`${resource} requires a persisted Core API identifier.`);
  }
  return id;
}
