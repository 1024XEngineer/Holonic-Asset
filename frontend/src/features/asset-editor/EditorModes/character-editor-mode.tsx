import { useEffect, useState } from "react";

import {
  AnimatedSpriteCanvas,
  getAnimatedSpriteNodeLabel,
  type AnimatedSpriteCanvasEvent,
  type AnimatedSpriteNodeId,
  type AnimatedSpriteCanvasSelection,
} from "../Canvas/AnimatedSpriteCanvas";
import type {
  EditorCanvasPosition,
  EditorCharacterAnimation,
  EditorCharacterSpriteSheet,
  GenerateAnimationRequest,
} from "@/model";

import { AssetTree } from "../AssetTree/asset-tree";
import { Inspector } from "../Inspector/inspector";
import type { EditorModeProps } from "./types";

export function CharacterEditorMode({
  prompt,
  history,
  characterPrototype,
  characterAnimations,
  characterNodePositions,
  onAction,
  onCharacterPositionChange,
  onCharacterAnimationGenerate,
  onCharacterAnimationRename,
  onCharacterAnimationDelete,
  isGeneratingAnimation,
  onPromptChange,
  renderHeader,
}: EditorModeProps & {
  characterPrototype: EditorCharacterSpriteSheet;
  characterAnimations: EditorCharacterAnimation[];
  characterNodePositions: Record<string, EditorCanvasPosition>;
  onCharacterPositionChange: (
    nodeId: string,
    position: EditorCanvasPosition,
  ) => void;
  onCharacterAnimationGenerate: (request: GenerateAnimationRequest) => void;
  onCharacterAnimationRename: (animationId: string, label: string) => void;
  onCharacterAnimationDelete: (animationId: string) => void;
  isGeneratingAnimation: boolean;
}) {
  const [canvasSelection, setCanvasSelection] =
    useState<AnimatedSpriteCanvasSelection>({
      nodeIds: [],
      frames: [],
    });
  useEffect(() => {
    const validNodeIds = new Set<string>([
      "prototype",
      ...characterAnimations.map((animation) => animation.id),
    ]);
    setCanvasSelection((current) => {
      const nodeIds = current.nodeIds.filter((node) => validNodeIds.has(node));
      const frames = current.frames.filter(
        (frame) =>
          validNodeIds.has(frame.nodeId) && nodeIds.includes(frame.nodeId),
      );
      if (
        nodeIds.length === current.nodeIds.length &&
        frames.length === current.frames.length
      ) {
        return current;
      }
      return { nodeIds, frames };
    });
  }, [characterAnimations]);
  const selection = canvasSelection.nodeIds.length
    ? canvasSelection.nodeIds
        .map((node) => getAnimatedSpriteNodeLabel(node, characterAnimations))
        .join(", ")
    : "Nothing selected";
  const selectNode = (nodeId: AnimatedSpriteNodeId) => {
    setCanvasSelection({ nodeIds: [nodeId], frames: [] });
  };
  const selectFrame = (nodeId: AnimatedSpriteNodeId, index: number) => {
    setCanvasSelection({
      nodeIds: [nodeId],
      frames: [{ nodeId, index }],
    });
  };
  const handleCanvasEvent = (event: AnimatedSpriteCanvasEvent) => {
    if (event.type === "selection.changed") {
      setCanvasSelection(event.selection);
      return;
    }

    onCharacterPositionChange(event.nodeId, event.position);
  };

  return (
    <>
      {renderHeader(selection)}
      <div className="flex min-h-0 flex-1 flex-col overflow-hidden lg:flex-row">
        <AssetTree
          animations={characterAnimations}
          selectedNode={canvasSelection.nodeIds[0] ?? null}
          selectedFrames={canvasSelection.frames}
          onSelect={selectNode}
          onSelectFrame={selectFrame}
          onGenerateAnimation={onCharacterAnimationGenerate}
          onRenameAnimation={onCharacterAnimationRename}
          onDeleteAnimation={onCharacterAnimationDelete}
          isGeneratingAnimation={isGeneratingAnimation}
        />
        <AnimatedSpriteCanvas
          model={{
            prototype: characterPrototype,
            animations: characterAnimations,
            nodePositions: characterNodePositions,
            selection: canvasSelection,
          }}
          onEvent={handleCanvasEvent}
        />
        <Inspector
          selectedNodes={canvasSelection.nodeIds}
          selectedFrames={canvasSelection.frames}
          prompt={prompt}
          onPromptChange={onPromptChange}
          onAction={onAction}
          history={history}
          animations={characterAnimations}
        />
      </div>
    </>
  );
}
