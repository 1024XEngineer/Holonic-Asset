import { useEffect, useMemo, useState } from "react";

import type { AssetWorkspaceData, TilesetItem, UISetComponent } from "@/model";

import {
  TilesetCanvas,
  useTilesetCanvasStateMachine,
} from "../Canvas/TilesetCanvas";
import { UISetCanvas } from "../Canvas/UISetCanvas";
import {
  EditorModeFrame,
  type EditorModeFrameProps,
} from "./editor-mode-frame";

type AssetCanvasEditorModeProps = {
  data: AssetWorkspaceData;
  onBack: () => void;
};

export function AssetCanvasEditorMode({
  data,
  onBack,
}: AssetCanvasEditorModeProps) {
  const frameProps = {
    assetName: data.asset.name,
    version: data.asset.version,
    projectName: data.projectName,
    onBack,
  };

  switch (data.record.mode) {
    case "tileset":
      return (
        <TilesetEditor
          {...frameProps}
          gridSize={data.record.tileset.gridSize}
          items={data.record.tileset.items}
        />
      );
    case "uiset":
      return (
        <UISetEditor
          {...frameProps}
          components={data.record.uiset.components}
        />
      );
    default:
      return null;
  }
}

function TilesetEditor({
  gridSize,
  items,
  ...frameProps
}: Omit<EditorModeFrameProps, "assetKind" | "children"> & {
  gridSize: number;
  items: TilesetItem[];
}) {
  const canvas = useTilesetCanvasStateMachine(items, gridSize);

  return (
    <EditorModeFrame {...frameProps} assetKind="tileset">
      <TilesetCanvas
        model={{
          gridSize,
          items,
          selectedCellIndexes: canvas.selectedCellIndexes,
        }}
        onEvent={canvas.send}
      />
    </EditorModeFrame>
  );
}

function UISetEditor({
  components,
  ...frameProps
}: Omit<EditorModeFrameProps, "assetKind" | "children"> & {
  components: UISetComponent[];
}) {
  const componentIds = useMemo(
    () => components.map((component) => component.id),
    [components],
  );
  const [selectedComponentIds, setSelectedComponentIds] = useState<string[]>(
    [],
  );

  useEffect(() => {
    setSelectedComponentIds((current) => {
      const next = current.filter((componentId) =>
        componentIds.includes(componentId),
      );
      return next.length === current.length ? current : next;
    });
  }, [componentIds]);

  return (
    <EditorModeFrame {...frameProps} assetKind="uiset">
      <UISetCanvas
        model={{ components, selectedComponentIds }}
        onEvent={({ componentId }) =>
          setSelectedComponentIds((current) =>
            current.includes(componentId)
              ? current.filter((id) => id !== componentId)
              : [...current, componentId],
          )
        }
      />
    </EditorModeFrame>
  );
}
