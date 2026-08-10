import type { AssetRevision, CharacterAnimation } from "@/model";

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

export type InspectorSubmitRequest = {
  prompt: string;
  reference?: InspectorReference;
  target: {
    nodeIds: AnimatedSpriteNodeId[];
    frames: InspectorFrameSelection[];
  };
};

export type InspectorProps = {
  selectedNodes: AnimatedSpriteNodeId[];
  selectedFrames: InspectorFrameSelection[];
  prompt: string;
  onPromptChange: (value: string) => void;
  history: AssetRevision[];
  animations: CharacterAnimation[];
  onSubmit: (request: InspectorSubmitRequest) => void | Promise<void>;
  onClearSelection: () => void;
  isSubmitting?: boolean;
  submitError?: string | null;
};

export type InspectorEditProps = Omit<InspectorProps, "history">;

export type InspectorTargetSummary = {
  label: string;
  detail: string;
};
