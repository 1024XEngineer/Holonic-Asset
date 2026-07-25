import type { EditorSceneryLayer } from "@/types/record";
import {
  SceneryCanvas,
  type SceneryCanvasEvent,
} from "@/modules/scenery-canvas";

import { SceneryLayerTree } from "../AssetTree/SceneryLayerTree";
import { useSceneryStageMachine } from "../Canvas/StateMachine/sceneryStageMachine";
import { Inspector } from "../Inspector/Inspector";
import type { EditorModeProps } from "./types";

export function SceneryEditorMode({
  prompt,
  history,
  onAction,
  onPromptChange,
  renderHeader,
  layers,
}: EditorModeProps & { layers: EditorSceneryLayer[] }) {
  const stage = useSceneryStageMachine(layers);
  const selection = stage.selectedLayers.length
    ? stage.selectedLayers.join(", ")
    : "Nothing selected";
  const handleCanvasEvent = (event: SceneryCanvasEvent) => {
    if (event.type === "layer.selection.toggled") {
      stage.toggleLayer(event.layerId);
    }
  };

  return (
    <>
      {renderHeader(selection)}
      <div className="flex min-h-0 flex-1 flex-col overflow-hidden lg:flex-row">
        <SceneryLayerTree
          layers={layers}
          selectedLayers={stage.selectedLayers}
          visibleLayers={stage.visibleLayers}
          onToggleLayer={stage.toggleLayer}
          onToggleVisibility={stage.toggleVisibility}
        />
        <SceneryCanvas
          model={{
            layers,
            selectedLayerIds: stage.selectedLayers,
            visibleLayerIds: stage.visibleLayers,
          }}
          onEvent={handleCanvasEvent}
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
