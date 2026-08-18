import type { CreatableAssetKind } from "@/model/asset";
import type { Perspective } from "@/model/project";
import type { ItemTile } from "@/model/item-tile";

export type CreationRequest<Reference = unknown> = {
  kind: CreatableAssetKind;
  name: string;
  prompt: string;
  canvasSize: string;
  dimensions?: { width: number; height: number };
  perspective?: Perspective;
  reference?: Reference;
  style?: string;
  aspectRatio?: string;
  layers?: { description: string }[];
  tiles?: {
    name: string;
    description: string;
    shape: ItemTile[];
  }[];
  components?: { name: string; description: string }[];
};

export type GenerationRun<Reference = unknown> = CreationRequest<Reference> & {
  id: string;
  projectId: string;
  assetId?: string;
  // The backend lifecycle is pending, processing, completed, failed, or cancelled.
  // This queue is a current-work projection: completed runs become assets and are
  // removed after the asset list refreshes; user-cancelled runs are removed once
  // cancellation succeeds. Only pending, processing, and actionable failures remain.
  status: "pending" | "processing" | "failed";
  error?: string;
};

export type GenerationInput<Reference = unknown> = {
  projectId: string;
  request: CreationRequest<Reference>;
};
