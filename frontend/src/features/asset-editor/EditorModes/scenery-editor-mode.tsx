import type { EditorSceneryLayer } from "@/model";
import {
  SceneryCanvas,
  useSceneryCanvasStateMachine,
} from "../Canvas/SceneryCanvas";

import { SceneryLayerTree } from "../AssetTree/scenery-layer-tree";
import { Inspector } from "../Inspector/inspector";
import type { EditorModeProps } from "./types";

export function SceneryEditorMode({
  prompt,
  history,
  onAction,
  onPromptChange,
  renderHeader,
  layers,
}: EditorModeProps & { layers: EditorSceneryLayer[] }) {
  const stage = useSceneryCanvasStateMachine(layers);
  const selection = stage.selectedLayers.length
    ? stage.selectedLayers.join(", ")
    : "Nothing selected";
  return (
    <>
      {renderHeader(selection)}
      <div className="flex min-h-0 flex-1 flex-col overflow-hidden lg:flex-row">
        <SceneryLayerTree
          layers={layers}
          selectedLayers={stage.selectedLayers}
          visibleLayers={stage.visibleLayers}
          onToggleLayer={(layerId) =>
            stage.send({ type: "layer.selection.toggled", layerId })
          }
          onToggleVisibility={(layerId) =>
            stage.send({ type: "layer.visibility.toggled", layerId })
          }
        />
        <SceneryCanvas
          model={{
            layers,
            selectedLayerIds: stage.selectedLayers,
            visibleLayerIds: stage.visibleLayers,
          }}
          onEvent={stage.send}
        />
        <Inspector
          selectedNodes={[]}
          selectedFrames={[]}
          selectedItems={stage.selectedLayers}
          prompt={prompt}
          onPromptChange={onPromptChange}
          onAction={onAction}
          history={history}
        />
      </div>
    </>
  );
}
