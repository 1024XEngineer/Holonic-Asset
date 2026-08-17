import { useTranslation } from "react-i18next";

import type {
  AssetKind,
  AssetWorkspaceData,
  SceneryLayer,
  TilesetItem,
} from "@/model";

import {
  SceneryCanvas,
  useSceneryCanvasStateMachine,
} from "../Canvas/SceneryCanvas";
import {
  TilesetCanvas,
  useTilesetCanvasStateMachine,
} from "../Canvas/TilesetCanvas";
import {
  EditorHeader,
  type EditorGenerationTask,
} from "../Header/editor-header";

type AssetCanvasEditorModeProps = {
  data: AssetWorkspaceData;
  onBack: () => void;
};

type CanvasEditorFrameProps = {
  assetKind: AssetKind;
  assetName: string;
  version: string;
  projectName: string;
  onBack: () => void;
  children: React.ReactNode;
};

const emptyGenerationTasks: EditorGenerationTask[] = [];
const noAction = () => undefined;

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
    case "scenery":
      return (
        <SceneryEditor {...frameProps} layers={data.record.scenery.layers} />
      );
    case "tileset":
      return (
        <TilesetEditor
          {...frameProps}
          gridSize={data.record.tileset.gridSize}
          items={data.record.tileset.items}
        />
      );
    default:
      return null;
  }
}

function SceneryEditor({
  layers,
  ...frameProps
}: Omit<CanvasEditorFrameProps, "assetKind" | "children"> & {
  layers: SceneryLayer[];
}) {
  const canvas = useSceneryCanvasStateMachine(layers);

  return (
    <CanvasEditorFrame {...frameProps} assetKind="scenery">
      <SceneryCanvas
        model={{
          layers,
          selectedLayerIds: canvas.selectedLayerIds,
          visibleLayerIds: canvas.visibleLayerIds,
        }}
        onEvent={canvas.send}
      />
    </CanvasEditorFrame>
  );
}

function TilesetEditor({
  gridSize,
  items,
  ...frameProps
}: Omit<CanvasEditorFrameProps, "assetKind" | "children"> & {
  gridSize: number;
  items: TilesetItem[];
}) {
  const canvas = useTilesetCanvasStateMachine(items, gridSize);

  return (
    <CanvasEditorFrame {...frameProps} assetKind="tileset">
      <TilesetCanvas
        model={{
          gridSize,
          items,
          selectedCellIndexes: canvas.selectedCellIndexes,
        }}
        onEvent={canvas.send}
      />
    </CanvasEditorFrame>
  );
}

function CanvasEditorFrame({
  assetKind,
  assetName,
  version,
  projectName,
  onBack,
  children,
}: CanvasEditorFrameProps) {
  const { t } = useTranslation("editor");
  return (
    <>
      <EditorHeader
        assetKind={assetKind}
        assetName={assetName}
        version={version}
        projectName={projectName}
        status={t("previewReady")}
        canUndo={false}
        canRedo={false}
        isDirty={false}
        isSaving={false}
        generationTasks={emptyGenerationTasks}
        onBack={onBack}
        onUndo={noAction}
        onRedo={noAction}
        onSave={noAction}
      />
      <div className="flex min-h-0 flex-1 overflow-hidden">{children}</div>
    </>
  );
}
