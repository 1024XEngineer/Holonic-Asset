import { useEffect, useState } from "react";

import type {
  AssetCanvasPosition,
  AssetRevision,
  CharacterAnimation,
  CharacterSpriteSheet,
  GenerateAnimationRequest,
  SpriteAssetKind,
} from "@/model";

import { AssetTree } from "../AssetTree/asset-tree";
import {
  AnimatedSpriteCanvas,
  type AnimatedSpriteCanvasEvent,
  type AnimatedSpriteCanvasSelection,
  type AnimatedSpriteNodeId,
} from "../Canvas/AnimatedSpriteCanvas";
import {
  EditorHeader,
  type EditorGenerationTask,
} from "../Header/editor-header";
import { Inspector } from "../Inspector/inspector";

export function SpriteEditorMode({
  assetKind,
  assetName,
  version,
  projectName,
  prototype,
  animations,
  nodePositions,
  prompt,
  history,
  status,
  canUndo,
  canRedo,
  isDirty,
  isSaving,
  isGeneratingAnimation,
  generationTasks,
  onBack,
  onUndo,
  onRedo,
  onSave,
  onPromptChange,
  onPositionChange,
  onAnimationGenerate,
  onAnimationRename,
  onAnimationDelete,
}: {
  assetKind: SpriteAssetKind;
  assetName: string;
  version: string;
  projectName: string;
  prototype: CharacterSpriteSheet;
  animations: CharacterAnimation[];
  nodePositions: Record<string, AssetCanvasPosition>;
  prompt: string;
  history: AssetRevision[];
  status: string;
  canUndo: boolean;
  canRedo: boolean;
  isDirty: boolean;
  isSaving: boolean;
  isGeneratingAnimation: boolean;
  generationTasks: EditorGenerationTask[];
  onBack: () => void;
  onUndo: () => void;
  onRedo: () => void;
  onSave: () => void;
  onPromptChange: (value: string) => void;
  onPositionChange: (nodeId: string, position: AssetCanvasPosition) => void;
  onAnimationGenerate: (request: GenerateAnimationRequest) => void;
  onAnimationRename: (animationId: string, label: string) => void;
  onAnimationDelete: (animationId: string) => void;
}) {
  const [selection, setSelection] = useState<AnimatedSpriteCanvasSelection>({
    nodeIds: [],
    frames: [],
  });

  useEffect(() => {
    const validNodeIds = new Set([
      "prototype",
      ...animations.map((animation) => animation.id),
    ]);
    setSelection((current) => {
      const nodeIds = current.nodeIds.filter((nodeId) =>
        validNodeIds.has(nodeId),
      );
      const frames = current.frames.filter(
        (frame) =>
          validNodeIds.has(frame.nodeId) && nodeIds.includes(frame.nodeId),
      );
      return nodeIds.length === current.nodeIds.length &&
        frames.length === current.frames.length
        ? current
        : { nodeIds, frames };
    });
  }, [animations]);

  const selectNode = (nodeId: AnimatedSpriteNodeId) => {
    setSelection({ nodeIds: [nodeId], frames: [] });
  };
  const selectFrame = (nodeId: AnimatedSpriteNodeId, index: number) => {
    setSelection({ nodeIds: [nodeId], frames: [{ nodeId, index }] });
  };
  const handleCanvasEvent = (event: AnimatedSpriteCanvasEvent) => {
    if (event.type === "selection.changed") setSelection(event.selection);
    else onPositionChange(event.nodeId, event.position);
  };

  return (
    <>
      <EditorHeader
        assetKind={assetKind}
        assetName={assetName}
        version={version}
        projectName={projectName}
        status={status}
        canUndo={canUndo}
        canRedo={canRedo}
        isDirty={isDirty}
        isSaving={isSaving}
        generationTasks={generationTasks}
        onBack={onBack}
        onUndo={onUndo}
        onRedo={onRedo}
        onSave={onSave}
      />
      <div className="flex min-h-0 flex-1 flex-col overflow-hidden lg:flex-row">
        <AssetTree
          animations={animations}
          selectedNode={selection.nodeIds[0] ?? null}
          selectedFrames={selection.frames}
          onSelect={selectNode}
          onSelectFrame={selectFrame}
          onGenerateAnimation={onAnimationGenerate}
          onRenameAnimation={onAnimationRename}
          onDeleteAnimation={onAnimationDelete}
          isGeneratingAnimation={isGeneratingAnimation}
        />
        <AnimatedSpriteCanvas
          model={{
            prototype,
            animations,
            nodePositions,
            selection,
          }}
          onEvent={handleCanvasEvent}
        />
        <Inspector
          selectedNodes={selection.nodeIds}
          selectedFrames={selection.frames}
          prompt={prompt}
          onPromptChange={onPromptChange}
          history={history}
          animations={animations}
        />
      </div>
    </>
  );
}
