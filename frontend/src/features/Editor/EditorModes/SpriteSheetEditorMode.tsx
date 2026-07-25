import type { SpriteSheetRecordContent } from "@/types/record";

import { StaticAssetTree } from "../AssetTree/StaticAssetTree";
import { SpriteSheetStage } from "../Canvas/SpriteSheetStage";
import { useSpriteSheetStageMachine } from "../Canvas/StateMachine/spriteSheetStageMachine";
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
  const stage = useSpriteSheetStageMachine(items);
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
          isTileSelected={stage.isTileSelected}
          onToggleItem={stage.toggleItem}
          onToggleTile={stage.toggleTile}
        />
        <SpriteSheetStage
          gridSize={spriteSheet.gridSize}
          selectedCells={stage.selectedCells}
          onToggleCell={stage.toggleCell}
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
