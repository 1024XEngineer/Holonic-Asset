import { useState } from "react";

import type { UiEditorRecord } from "@/model";
import { UiCanvas } from "../Canvas/UiCanvas";

import { UiComponentTree } from "../AssetTree/UiComponentTree";
import { Inspector } from "../Inspector/Inspector";
import type { EditorModeProps } from "./types";

export function UiEditorMode({
  prompt,
  history,
  onAction,
  onPromptChange,
  renderHeader,
  ui,
}: EditorModeProps & { ui: UiEditorRecord["ui"] }) {
  const [selectedComponentIds, setSelectedComponentIds] = useState<string[]>(
    [],
  );
  const selectedLabels = ui.components
    .filter((component) => selectedComponentIds.includes(component.id))
    .map((component) => component.label);
  const selection = selectedLabels.length
    ? selectedLabels.join(", ")
    : "Nothing selected";
  const toggleComponent = (componentId: string) =>
    setSelectedComponentIds((current) =>
      current.includes(componentId)
        ? current.filter((id) => id !== componentId)
        : [...current, componentId],
    );

  return (
    <>
      {renderHeader(selection)}
      <div className="flex min-h-0 flex-1 flex-col overflow-hidden lg:flex-row">
        <UiComponentTree
          components={ui.components}
          selectedComponentIds={selectedComponentIds}
          onToggleComponent={toggleComponent}
        />
        <UiCanvas
          model={{ components: ui.components, selectedComponentIds }}
          onEvent={(event) => toggleComponent(event.componentId)}
        />
        <Inspector
          selectedNodes={[]}
          selectedFrames={[]}
          selectedItems={selectedLabels}
          prompt={prompt}
          onPromptChange={onPromptChange}
          onAction={onAction}
          history={history}
        />
      </div>
    </>
  );
}
