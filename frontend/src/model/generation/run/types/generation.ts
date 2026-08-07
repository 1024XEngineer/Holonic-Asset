import type { CreatableAssetKind } from "@/model/asset";
import type { Perspective } from "@/model/project";

export type TileShapeCell = [column: number, row: number];

export type CreationRequest<Reference = unknown> = {
  kind: CreatableAssetKind;
  name: string;
  prompt: string;
  canvasSize: string;
  perspective?: Perspective;
  directionCount?: "1" | "4" | "8";
  reference?: Reference;
  useProjectContext: boolean;
  style?: string;
  aspectRatio?: string;
  layers?: { description: string }[];
  tiles?: {
    description: string;
    reference?: Reference;
    shape: TileShapeCell[];
  }[];
  components?: { name: string; description: string; isCustom: boolean }[];
};

export type GenerationRun<Reference = unknown> = CreationRequest<Reference> & {
  id: string;
  projectId: string;
  // The backend lifecycle is pending, processing, completed, failed, or cancelled.
  // This queue is a current-work projection: completed runs become assets and are
  // removed after the asset list refreshes; user-cancelled runs are removed once
  // cancellation succeeds. Only pending, processing, and actionable failures remain.
  status: "pending" | "processing" | "failed";
};

export type GenerationInput<Reference = unknown> = {
  projectId: string;
  request: CreationRequest<Reference>;
};
