import type { AssetKind, ProjectAsset } from "@/model/asset";
import { getMockProject } from "@/model/project/mock";
import type { GenerationRun } from "../types";

export type MockGeneratedAsset = {
  asset: ProjectAsset;
  kind: AssetKind;
};

export async function completeMockGeneration(
  run: GenerationRun,
): Promise<MockGeneratedAsset> {
  await new Promise<void>((resolve) => {
    setTimeout(resolve, 700);
  });
  const project = await getMockProject(run.projectId);

  return {
    kind: run.kind,
    asset: {
      id: `asset-${run.id}`,
      name: run.name,
      description: run.prompt,
      version: "v1",
      canvasSize: run.canvasSize,
      perspective: run.perspective ?? project.perspective,
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
  };
}
