import type {
  AssetGroup,
  AssetKind,
  ProjectAsset,
} from "@/features/assets/types";

import type { CreationRequest, GenerationRun } from "./generation";

export type GenerationInput = { projectId: string; request: CreationRequest };
export type GenerationLifecycleUpdate =
  | { kind: "run-upserted"; run: GenerationRun }
  | { kind: "run-removed"; projectId: string; runId: string };
export type GenerationLifecycleResult = {
  assetGroups?: AssetGroup[];
  run: GenerationRun;
};
export type GenerationLifecycleAdapter = {
  createRun: (run: Omit<GenerationRun, "id" | "status">) => GenerationRun;
  updateRun: (run: GenerationRun) => GenerationRun;
  removeRun: (runId: string) => void;
  completeGeneration: (
    run: GenerationRun,
  ) => Promise<{ asset: ProjectAsset; kind: AssetKind }>;
  hasProject: (projectId: string) => boolean;
  addAsset: (
    projectId: string,
    kind: AssetKind,
    asset: ProjectAsset,
  ) => Promise<AssetGroup[]>;
};

export function createGenerationLifecycle(adapter: GenerationLifecycleAdapter) {
  return {
    async enqueue(
      { projectId, request }: GenerationInput,
      onUpdate: (update: GenerationLifecycleUpdate) => void,
    ): Promise<GenerationLifecycleResult> {
      const queuedRun = adapter.createRun({ ...request, projectId });
      onUpdate({ kind: "run-upserted", run: queuedRun });
      const processingRun = adapter.updateRun({
        ...queuedRun,
        status: "processing",
      });
      onUpdate({ kind: "run-upserted", run: processingRun });
      const removeRun = () => {
        adapter.removeRun(processingRun.id);
        onUpdate({
          kind: "run-removed",
          projectId,
          runId: processingRun.id,
        });
      };
      try {
        const generated = await adapter.completeGeneration(processingRun);
        if (!adapter.hasProject(projectId)) {
          removeRun();
          return { run: processingRun };
        }
        const assetGroups = await adapter.addAsset(
          projectId,
          generated.kind,
          generated.asset,
        );
        removeRun();
        return { assetGroups, run: processingRun };
      } catch (error) {
        if (!adapter.hasProject(projectId)) {
          removeRun();
          return { run: processingRun };
        }
        const failedRun = adapter.updateRun({
          ...processingRun,
          status: "failed",
        });
        onUpdate({ kind: "run-upserted", run: failedRun });
        throw error;
      }
    },
  };
}
