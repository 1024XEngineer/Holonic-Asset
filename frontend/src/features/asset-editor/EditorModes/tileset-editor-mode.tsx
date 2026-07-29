import type { TilesetEditorRecord } from "@/model";
import {
  TilesetCanvas,
  useTilesetCanvasStateMachine,
} from "../Canvas/TilesetCanvas";

import { TilesetAssetTree } from "../AssetTree/tileset-asset-tree";
import { Inspector } from "../Inspector/inspector";
import type { EditorModeProps } from "./types";

export function TilesetEditorMode({
  prompt,
  history,
  onAction,
  onPromptChange,
  renderHeader,
  tileset,
}: EditorModeProps & {
  tileset: TilesetEditorRecord["tileset"];
}) {
  const items = tileset.items;
  const stage = useTilesetCanvasStateMachine(items, tileset.gridSize);
  const selection = stage.selectedLabels.length
    ? stage.selectedLabels.join(", ")
    : "Nothing selected";
  return (
    <>
      {renderHeader(selection)}
      <div className="flex min-h-0 flex-1 flex-col overflow-hidden lg:flex-row">
        <TilesetAssetTree
          items={items}
          selectedItems={stage.selectedItems}
          isCellSelected={stage.isCellSelected}
          onToggleItem={(itemId) => stage.send({ type: "item.toggle", itemId })}
          onToggleCell={(itemId, cellIndex) =>
            stage.send({ type: "item-cell.toggle", itemId, cellIndex })
          }
        />
        <TilesetCanvas
          model={{
            gridSize: tileset.gridSize,
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
