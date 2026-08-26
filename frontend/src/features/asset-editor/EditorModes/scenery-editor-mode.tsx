import { useAssetExport, type AssetWorkspaceData } from "@/model";

import { SceneryAssetTree } from "../AssetTree/scenery-asset-tree";
import {
  SceneryCanvas,
  useSceneryCanvasStateMachine,
} from "../Canvas/SceneryCanvas";
import { Inspector } from "../Inspector/inspector";
import { EditorModeFrame } from "./editor-mode-frame";

export type SceneryEditorModeProps = {
  data: AssetWorkspaceData;
  onBack: () => void;
};

const emptySceneryLayers = [] as const;

export function SceneryEditorMode({ data, onBack }: SceneryEditorModeProps) {
  const { asset, projectName, record } = data;
  const { exportAsset, isExporting } = useAssetExport();
  const canvas = useSceneryCanvasStateMachine(
    record.mode === "scenery" ? record.scenery.layers : emptySceneryLayers,
  );
  if (record.mode !== "scenery") return null;
  const { layers, dimensions } = record.scenery;
  const handleExport = async () => {
    const assetId = Number(asset.id);
    if (!Number.isSafeInteger(assetId) || assetId <= 0) return;
    try {
      await exportAsset(assetId);
    } catch {
      // The export hook exposes the failure state for the header lifecycle.
    }
  };
  const selectedLayerId = canvas.selectedLayerIds[0] ?? null;
  const selectedLayer =
    layers.find((layer) => layer.id === selectedLayerId) ?? null;

  return (
    <EditorModeFrame
      assetKind="scenery"
      assetName={asset.name}
      version={asset.version}
      projectName={projectName}
      onBack={onBack}
      canExport
      isExporting={isExporting}
      onExport={handleExport}
    >
      <SceneryAssetTree
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
        history={asset.history}
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
    </EditorModeFrame>
  );
}
