import type { SpriteSheetRecordContent } from "@/features/assets/domain";
import {
  SpriteSheetCanvas,
  useSpriteSheetCanvasStateMachine,
} from "../../modules/sprite-sheet-canvas";

import { StaticAssetTree } from "../AssetTree/StaticAssetTree";
import { Inspector } from "../Inspector/Inspector";
import type { EditorModeProps } from "./types";

export function SpriteSheetEditorMode({
  prompt,
  history,
  onAction,
  onPromptChange,
  renderHeader,
  spriteSheet,
}: EditorModeProps & {
  spriteSheet: SpriteSheetRecordContent["spriteSheet"];
}) {
  const items = spriteSheet.items;
  const stage = useSpriteSheetCanvasStateMachine(items, spriteSheet.gridSize);
  const selection = stage.selectedLabels.length
    ? stage.selectedLabels.join(", ")
    : "Nothing selected";
  return (
    <>
      {renderHeader(selection)}
      <div className="flex min-h-0 flex-1 flex-col overflow-hidden lg:flex-row">
        <StaticAssetTree
          items={items}
          selectedItems={stage.selectedItems}
          isCellSelected={stage.isCellSelected}
          onToggleItem={(itemId) => stage.send({ type: "item.toggle", itemId })}
          onToggleCell={(itemId, cellIndex) =>
            stage.send({ type: "item-cell.toggle", itemId, cellIndex })
          }
        />
        <SpriteSheetCanvas
          model={{
            gridSize: spriteSheet.gridSize,
            items,
            selectedCellIndexes: stage.selectedCells,
          }}
          onEvent={stage.send}
        />
        <Inspector
          selectedNodes={[]}
          selectedFrames={[]}
          selectedItems={stage.selectedLabels}
          prompt={prompt}
          onPromptChange={onPromptChange}
          onAction={onAction}
          history={history}
        />
      </div>
    </>
  );
}
