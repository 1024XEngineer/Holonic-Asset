import { useState } from "react";

import {
  CharacterCanvas,
  getCharacterNodeLabel,
  type CharacterCanvasEvent,
  type CharacterCanvasNodeId,
  type CharacterCanvasSelection,
} from "@/modules/character-canvas";
import type {
  EditorCanvasPosition,
  EditorCharacterAnimation,
} from "@/types/record";

import { AssetTree } from "../AssetTree/AssetTree";
import { Inspector } from "../Inspector/Inspector";
import type { EditorModeProps } from "./types";

export function CharacterEditorMode({
  prompt,
  history,
  characterAnimations,
  characterNodePositions,
  onAction,
  onCharacterPositionChange,
  onPromptChange,
  renderHeader,
}: EditorModeProps & {
  characterAnimations: EditorCharacterAnimation[];
  characterNodePositions: Record<string, EditorCanvasPosition>;
  onCharacterPositionChange: (
    nodeId: string,
    position: EditorCanvasPosition,
  ) => void;
}) {
  const [canvasSelection, setCanvasSelection] =
    useState<CharacterCanvasSelection>({
      nodeIds: [],
      frames: [],
    });
  const selection = canvasSelection.nodeIds.length
    ? canvasSelection.nodeIds
        .map((node) => getCharacterNodeLabel(node, characterAnimations))
        .join(", ")
    : "Nothing selected";
  const selectNode = (nodeId: CharacterCanvasNodeId) => {
    setCanvasSelection({ nodeIds: [nodeId], frames: [] });
  };
  const selectFrame = (nodeId: CharacterCanvasNodeId, index: number) => {
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
        />
        <CharacterCanvas
          model={{
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
        />
      </div>
    </>
  );
}
