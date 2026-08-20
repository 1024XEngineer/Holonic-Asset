import type { AssetWorkspaceData } from "@/model";

import { AssetTree } from "../AssetTree/asset-tree";
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
  const canvas = useSceneryCanvasStateMachine(
    record.mode === "scenery" ? record.scenery.layers : emptySceneryLayers,
  );
  if (record.mode !== "scenery") return null;
  const { layers, dimensions } = record.scenery;
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
    >
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
