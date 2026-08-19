import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";

import type {
  AssetKind,
  AssetRevision,
  SceneryCanvasDimensions,
  AssetWorkspaceData,
  SceneryLayer,
  TilesetItem,
  UISetComponent,
} from "@/model";

import {
  SceneryCanvas,
  useSceneryCanvasStateMachine,
} from "../Canvas/SceneryCanvas";
import {
  TilesetCanvas,
  useTilesetCanvasStateMachine,
} from "../Canvas/TilesetCanvas";
import { UISetCanvas } from "../Canvas/UISetCanvas";
import { AssetTree } from "../AssetTree/asset-tree";
import {
  EditorHeader,
  type EditorGenerationTask,
} from "../Header/editor-header";
import { Inspector } from "../Inspector/inspector";

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
        <SceneryEditor
          {...frameProps}
          {...data.record.scenery}
          history={data.asset.history}
        />
      );
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

function SceneryEditor({
  layers,
  dimensions,
  history,
  ...frameProps
}: Omit<CanvasEditorFrameProps, "assetKind" | "children"> & {
  layers: SceneryLayer[];
  dimensions?: SceneryCanvasDimensions;
  history: AssetRevision[];
}) {
  const canvas = useSceneryCanvasStateMachine(layers);
  const selectedLayerId = canvas.selectedLayerIds[0] ?? null;
  const selectedLayer =
    layers.find((layer) => layer.id === selectedLayerId) ?? null;

  return (
    <CanvasEditorFrame {...frameProps} assetKind="scenery">
      <AssetTree
        kind="scenery"
        layers={layers}
        selectedLayerId={selectedLayerId}
        visibleLayerIds={canvas.visibleLayerIds}
        onSelect={(layerId) =>
          canvas.send({ type: "layer.selection.toggled", layerId })
        }
        onToggleVisibility={(layerId) =>
          canvas.send({ type: "layer.visibility.toggled", layerId })
        }
      />
      <SceneryCanvas
        model={{
          layers,
          dimensions,
          selectedLayerIds: canvas.selectedLayerIds,
          visibleLayerIds: canvas.visibleLayerIds,
        }}
        onEvent={canvas.send}
      />
      <Inspector
        kind="scenery"
        layer={selectedLayer}
        dimensions={dimensions}
        history={history}
        visible={
          selectedLayer
            ? canvas.visibleLayerIds.includes(selectedLayer.id)
            : false
        }
        onToggleVisibility={() => {
          if (selectedLayer) {
            canvas.send({
              type: "layer.visibility.toggled",
              layerId: selectedLayer.id,
            });
          }
        }}
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

function UISetEditor({
  components,
  ...frameProps
}: Omit<CanvasEditorFrameProps, "assetKind" | "children"> & {
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
    <CanvasEditorFrame {...frameProps} assetKind="uiset">
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
      <div className="flex min-h-0 flex-1 flex-col overflow-hidden lg:flex-row">
        {children}
      </div>
    </>
  );
}
