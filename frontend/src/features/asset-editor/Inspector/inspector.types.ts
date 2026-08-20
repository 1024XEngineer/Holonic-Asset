import type {
  AssetRevision,
  CharacterAnimation,
  CharacterSpriteSheet,
  SceneryCanvasDimensions,
  SceneryLayer,
} from "@/model";
import { z } from "zod";

import type { AnimatedSpriteNodeId } from "../Canvas/AnimatedSpriteCanvas";

export type InspectorFrameSelection = {
  nodeId: AnimatedSpriteNodeId;
  index: number;
};

export type InspectorReference = {
  fileName: string;
  mimeType: string;
  dataUrl: string;
};

const inspectorReferenceSchema = z.object({
  fileName: z.string().min(1),
  mimeType: z.string().min(1),
  dataUrl: z.string().min(1),
});

const inspectorFrameSelectionSchema = z.object({
  nodeId: z.string().min(1),
  index: z.int().nonnegative(),
});

export const inspectorPromptSchema = z.string().trim().min(1);

export const inspectorSubmitRequestSchema = z.object({
  prompt: inspectorPromptSchema,
  reference: inspectorReferenceSchema.optional(),
  target: z.object({
    nodeIds: z.array(z.string().min(1)),
    frames: z.array(inspectorFrameSelectionSchema),
  }),
});

export type InspectorSubmitRequest = z.infer<
  typeof inspectorSubmitRequestSchema
>;

export type SpriteInspectorProps = {
  kind: "sprite";
  selectedNodes: AnimatedSpriteNodeId[];
  selectedFrames: InspectorFrameSelection[];
  prompt: string;
  onPromptChange: (value: string) => void;
  history: AssetRevision[];
  animations: CharacterAnimation[];
  prototype: CharacterSpriteSheet;
  onSubmit: (request: InspectorSubmitRequest) => void | Promise<void>;
  onClearSelection: () => void;
  isSubmitting?: boolean;
};

export type SceneryInspectorProps = {
  kind: "scenery";
  layer: SceneryLayer | null;
  dimensions?: SceneryCanvasDimensions;
  history: AssetRevision[];
  visible: boolean;
  onToggleVisibility: () => void;
};

export type InspectorProps = SpriteInspectorProps | SceneryInspectorProps;

export type SpriteInspectorContentProps = Omit<
  SpriteInspectorProps,
  "history" | "kind"
>;

export type InspectorTargetSummary = {
  label: string;
  detail: string;
  thumbnail: {
    imageUrl: string;
    column: number;
    row: number;
    columns: number;
    rows: number;
  };
};
