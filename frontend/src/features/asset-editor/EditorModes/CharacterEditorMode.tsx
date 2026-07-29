import { useEffect, useState } from "react";

import {
  AnimatedSpriteCanvas,
  createDefaultAnimatedSpriteDirections,
  findAnimatedSpriteAnimationGroup,
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

import { AssetTree } from "../AssetTree/AssetTree";
import { Inspector } from "../Inspector/Inspector";
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
  const [activeDirections, setActiveDirections] = useState<
    Record<string, AnimatedSpriteNodeId>
  >(() => createDefaultAnimatedSpriteDirections(characterAnimations));

  useEffect(() => {
    const validNodeIds = new Set<string>([
      "prototype",
      ...characterAnimations.flatMap((animation) =>
        animation.kind === "group"
          ? [
              animation.id,
              ...animation.directions.map((direction) => direction.id),
            ]
          : [animation.id],
      ),
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
    setActiveDirections((current) => {
      const next = Object.fromEntries(
        characterAnimations
          .filter((animation) => animation.kind === "group")
          .map((animation) => [
            animation.id,
            animation.directions.some(
              (direction) => direction.id === current[animation.id],
            )
              ? current[animation.id]
              : animation.directions[0].id,
          ]),
      );
      return JSON.stringify(next) === JSON.stringify(current) ? current : next;
    });
  }, [characterAnimations]);
  const selection = canvasSelection.nodeIds.length
    ? canvasSelection.nodeIds
        .map((node) => getAnimatedSpriteNodeLabel(node, characterAnimations))
        .join(", ")
    : "Nothing selected";
  const selectNode = (nodeId: AnimatedSpriteNodeId) => {
    const group = findAnimatedSpriteAnimationGroup(nodeId, characterAnimations);
    if (group) {
      setActiveDirections((current) => ({
        ...current,
        [group.id]: nodeId,
      }));
    }
    setCanvasSelection({ nodeIds: [nodeId], frames: [] });
  };
  const selectFrame = (nodeId: AnimatedSpriteNodeId, index: number) => {
    const group = findAnimatedSpriteAnimationGroup(nodeId, characterAnimations);
    if (group) {
      setActiveDirections((current) => ({
        ...current,
        [group.id]: nodeId,
      }));
    }
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

    if (event.type === "direction.changed") {
      setActiveDirections((current) => ({
        ...current,
        [event.nodeId]: event.directionId,
      }));
      setCanvasSelection({ nodeIds: [event.directionId], frames: [] });
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
            activeDirections,
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
