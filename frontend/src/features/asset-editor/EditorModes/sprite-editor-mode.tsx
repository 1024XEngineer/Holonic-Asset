import { useEffect, useState } from "react";

import { AssetTree } from "../AssetTree/asset-tree";
import {
  AnimatedSpriteCanvas,
  type AnimatedSpriteCanvasEvent,
  type AnimatedSpriteCanvasSelection,
  type AnimatedSpriteNodeId,
} from "../Canvas/AnimatedSpriteCanvas";
import { EditorHeader } from "../Header/editor-header";
import { Inspector } from "../Inspector/inspector";
import type { SpriteEditorModeProps } from "./sprite-editor-mode.types";

export function SpriteEditorMode({
  header,
  sprite,
  tree,
  inspector,
  generationReview,
}: SpriteEditorModeProps) {
  const { animations } = sprite;
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
    switch (event.type) {
      case "selection.changed":
        setSelection(event.selection);
        return;
      case "node-position.committed":
        sprite.onPositionChange(event.nodeId, event.position);
        return;
      case "generation-review.resolved":
        if (event.applied) generationReview?.onApply();
        else generationReview?.onDeny();
    }
  };
  const clearInspectorSelection = () => {
    setSelection({ nodeIds: [], frames: [] });
  };

  return (
    <>
      <EditorHeader {...header} />
      <div className="flex min-h-0 flex-1 flex-col overflow-hidden lg:flex-row">
        <AssetTree
          kind="sprite"
          animations={animations}
          perspective={sprite.perspective}
          selectedNode={selection.nodeIds[0] ?? null}
          selectedFrames={selection.frames}
          onSelect={selectNode}
          onSelectFrame={selectFrame}
          onGenerateAnimation={tree.onAnimationGenerate}
          onRenameAnimation={tree.onAnimationRename}
          onDeleteAnimation={tree.onAnimationDelete}
          isGeneratingAnimation={tree.isGeneratingAnimation}
        />
        <AnimatedSpriteCanvas
          model={{
            prototype: sprite.prototype,
            animations,
            nodePositions: sprite.nodePositions,
            selection,
            ...(generationReview ? { review: generationReview } : {}),
          }}
          onEvent={handleCanvasEvent}
        />
        <Inspector
          kind="sprite"
          {...inspector}
          selectedNodes={selection.nodeIds}
          selectedFrames={selection.frames}
          onClearSelection={clearInspectorSelection}
          animations={animations}
          prototype={sprite.prototype}
        />
      </div>
    </>
  );
}
