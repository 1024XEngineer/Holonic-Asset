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
  // Completed and cancelled runs leave this current-work projection. Awaiting
  // results stay visible until the editor explicitly applies or discards them.
  status: "pending" | "processing" | "awaiting_application" | "failed";
  error?: string;
};

export type GenerationInput<Reference = unknown> = {
  projectId: string;
  request: CreationRequest<Reference>;
};
