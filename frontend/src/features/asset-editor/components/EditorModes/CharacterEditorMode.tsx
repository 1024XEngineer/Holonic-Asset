import { useState } from "react";

import {
  CharacterCanvas,
  createDefaultCharacterDirections,
  findCharacterAnimationGroup,
  getCharacterNodeLabel,
  type CharacterCanvasEvent,
  type CharacterCanvasNodeId,
  type CharacterCanvasSelection,
} from "../../modules/character-canvas";
import type {
  EditorCanvasPosition,
  EditorCharacterAnimation,
  EditorCharacterSpriteSheet,
} from "../../domain";

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
  onCharacterAnimationCreate,
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
  onCharacterAnimationCreate: (label: string) => void;
}) {
  const [canvasSelection, setCanvasSelection] =
    useState<CharacterCanvasSelection>({
      nodeIds: [],
      frames: [],
    });
  const [activeDirections, setActiveDirections] = useState<
    Record<string, CharacterCanvasNodeId>
  >(() => createDefaultCharacterDirections(characterAnimations));
  const selection = canvasSelection.nodeIds.length
    ? canvasSelection.nodeIds
        .map((node) => getCharacterNodeLabel(node, characterAnimations))
        .join(", ")
    : "Nothing selected";
  const selectNode = (nodeId: CharacterCanvasNodeId) => {
    const group = findCharacterAnimationGroup(nodeId, characterAnimations);
    if (group) {
      setActiveDirections((current) => ({
        ...current,
        [group.id]: nodeId,
      }));
    }
    setCanvasSelection({ nodeIds: [nodeId], frames: [] });
  };
  const selectFrame = (nodeId: CharacterCanvasNodeId, index: number) => {
    const group = findCharacterAnimationGroup(nodeId, characterAnimations);
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
  const handleCanvasEvent = (event: CharacterCanvasEvent) => {
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
          onCreateAnimation={onCharacterAnimationCreate}
        />
        <CharacterCanvas
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
