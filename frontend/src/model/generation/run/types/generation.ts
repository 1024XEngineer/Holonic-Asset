import type { AssetTag, CreatableAssetKind } from "@/model/asset";
import type { Perspective } from "@/model/project";
import type { ItemTile } from "@/model/item-tile";

export type CreationRequest<CreatingReference = unknown> = {
  kind: CreatableAssetKind;
  name: string;
  prompt: string;
  canvasSize: string;
  dimensions?: { width: number; height: number };
  perspective?: Perspective;
  tags?: AssetTag[];
  creatingReference?: CreatingReference;
  style?: string;
  tiles?: {
    name: string;
    description: string;
    shape: ItemTile[];
  }[];
  components?: { name: string; description: string }[];
};

export type GenerationRun<CreatingReference = unknown> =
  CreationRequest<CreatingReference> & {
    id: string;
    projectId: string;
    assetId?: string;
    // Completed and cancelled runs leave this current-work projection. Awaiting
    // results stay visible until the editor explicitly applies or discards them.
    status: "pending" | "processing" | "awaiting_application" | "failed";
    error?: string;
  };

export type GenerationInput<CreatingReference = unknown> = {
  projectId: string;
  request: CreationRequest<CreatingReference>;
};
