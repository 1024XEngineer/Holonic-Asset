import type { CreatableAssetKind } from "../../asset";

export type CreationRequest<Reference = unknown> = {
  kind: CreatableAssetKind;
  name: string;
  prompt: string;
  canvasSize: string;
  perspective?: "top-down" | "side-on" | "isometric";
  directionCount?: "1" | "4" | "8";
  reference?: Reference;
  useProjectContext: boolean;
  style?: string;
  aspectRatio?: string;
  layers?: { description: string }[];
  tiles?: { description: string; reference?: Reference }[];
  components?: { name: string; description: string; isCustom: boolean }[];
};

export type GenerationRun<Reference = unknown> = CreationRequest<Reference> & {
  id: string;
  projectId: string;
  status: "queued" | "processing" | "failed";
};
