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
  const selectedItems = [
    ...stage.selectedItems,
    ...stage.selectedTiles.map((tile) => {
      const separator = tile.lastIndexOf(":");
      const item = tile.slice(0, separator);
      const tileIndex = Number(tile.slice(separator + 1));
      const itemData = items.find((candidate) => candidate.id === item);

      return item === "Canvas"
        ? `Tile ${tileIndex + 1}`
        : `${itemData?.label ?? item} / ${itemData?.tiles[tileIndex]?.label ?? `Tile ${tileIndex + 1}`}`;
    }),
  ];
  const selection = selectedItems.length
    ? selectedItems.join(", ")
    : "Nothing selected";

  return (
    <>
      {renderHeader(selection)}
      <div className="flex min-h-0 flex-1 flex-col overflow-hidden lg:flex-row">
        <StaticAssetTree
          items={items}
          selectedItems={stage.selectedItems}
          selectedTiles={stage.selectedTiles}
          onToggleItem={stage.toggleItem}
          onToggleTile={stage.toggleTile}
        />
        <SpriteSheetStage
          gridSize={spriteSheet.gridSize}
          items={items}
          selectedItems={stage.selectedItems}
          selectedTiles={stage.selectedTiles}
          onToggleTile={stage.toggleTile}
        />
        <Inspector
          selectedNodes={[]}
          selectedFrames={[]}
          selectedItems={selectedItems}
          prompt={prompt}
          onPromptChange={onPromptChange}
          onAction={onAction}
          history={history}
        />
      </div>
    </>
  );
}
