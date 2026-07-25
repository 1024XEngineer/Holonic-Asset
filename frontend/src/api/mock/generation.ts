import type { ProjectAsset } from "@/types/asset";
import type { AssetKind } from "@/types/asset-kind";
import type { GenerationRun } from "@/types/generation";

export type MockGeneratedAsset = {
  asset: ProjectAsset;
  kind: AssetKind;
};

export function completeMockGeneration(
  run: GenerationRun,
): Promise<MockGeneratedAsset> {
  return new Promise((resolve) => {
    setTimeout(() => {
      resolve({
        kind: run.kind,
        asset: {
          id: `asset-${run.id}`,
          name: run.name,
          description: run.prompt,
          version: "v1",
          canvasSize: run.canvasSize,
          perspective: run.perspective ?? "Not specified",
          tags: [run.kind, "generated"],
          history: [
            {
              id: `record-${run.id}-v1`,
              version: "v1",
              description: run.prompt,
              status: "ready",
              isCurrent: true,
            },
          ],
          animations: [],
        },
      });
    }, 700);
  });
}
