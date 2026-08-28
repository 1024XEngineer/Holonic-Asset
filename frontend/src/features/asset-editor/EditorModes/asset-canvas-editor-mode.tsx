import { useEffect, useMemo, useState } from "react";

import { useTranslation } from "react-i18next";

import type { AssetWorkspaceData, UISetComponent } from "@/model";

import { TilesetAssetTree } from "../AssetTree/tileset-asset-tree";
import {
  TilesetCanvas,
  type TilesetCanvasEvent,
  useTilesetCanvasStateMachine,
} from "../Canvas/TilesetCanvas";
import { UISetCanvas } from "../Canvas/UISetCanvas";
import { EditorHeader } from "../Header/editor-header";
import { Inspector } from "../Inspector/inspector";
import {
  MAX_TILESET_EDIT_TARGETS,
  resolveTilesetEditTarget,
} from "../tileset-edit-target";
import { useTilesetEditorWorkspace } from "../use-tileset-editor-workspace";
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
  switch (data.record.mode) {
    case "tileset":
      return <TilesetEditor data={data} onBack={onBack} />;
    case "uiset":
      return (
        <UISetEditor
          assetName={data.asset.name}
          version={data.asset.version}
          projectName={data.projectName}
          onBack={onBack}
          components={data.record.uiset.components}
        />
      );
    default:
      return null;
  }
}

function TilesetEditor({ data, onBack }: AssetCanvasEditorModeProps) {
  const editor = useTilesetEditorWorkspace({ data, onBack });
  return editor ? <ConnectedTilesetEditor editor={editor} /> : null;
}

function ConnectedTilesetEditor({
  editor,
}: {
  editor: NonNullable<ReturnType<typeof useTilesetEditorWorkspace>>;
}) {
  const { t } = useTranslation("editor");
  const canvas = useTilesetCanvasStateMachine(
    editor.sourceItems,
    editor.gridSize,
  );
  const resolution = resolveTilesetEditTarget({
    selectedCellIndexes: canvas.selectedCellIndexes,
    items: editor.sourceItems,
    gridSize: editor.gridSize,
  });
  const target = resolution.target
    ? { label: resolution.target.label, detail: t("tilesetTargetDetail") }
    : null;
  let targetError: string | null = null;
  switch (resolution.error) {
    case "too-many":
      targetError = t("tooManyTilesetTargets", {
        count: MAX_TILESET_EDIT_TARGETS,
      });
      break;
    case "multiple-items":
      targetError = t("tilesetTargetsMustShareItem");
      break;
    case "missing":
      targetError = t("selectTilesetTarget");
      break;
  }
  const handleCanvasEvent = (event: TilesetCanvasEvent) => {
    switch (event.type) {
      case "cell.selection.toggled":
        canvas.send(event);
        return;
      case "generation-review.resolved":
        editor.onResolveReview(event.applied);
    }
  };

  return (
    <>
      <EditorHeader {...editor.header} />
      <div className="flex min-h-0 flex-1 flex-col overflow-hidden lg:flex-row">
        <TilesetAssetTree
          items={editor.sourceItems}
          selectedItemIds={canvas.selectedItems}
          isTileSelected={canvas.isCellSelected}
          onToggleItem={(itemId) =>
            canvas.send({ type: "item.toggle", itemId })
          }
          onToggleTile={(itemId, itemCellIndex) =>
            canvas.send({ type: "item-cell.toggle", itemId, itemCellIndex })
          }
          onGenerateItem={editor.onGenerateItem}
          isGeneratingItem={editor.isGeneratingItem}
        />
        <TilesetCanvas
          model={{
            gridSize: editor.gridSize,
            items: editor.items,
            selectedCellIndexes: canvas.selectedCellIndexes,
            ...(editor.review ? { review: editor.review } : {}),
          }}
          onEvent={handleCanvasEvent}
        />
        <Inspector
          kind="tileset"
          prompt={editor.prompt}
          history={editor.history}
          target={target}
          targetError={targetError}
          isSubmitting={editor.isSubmitting}
          onPromptChange={editor.onPromptChange}
          onSubmit={async (request) => {
            if (!resolution.target) return false;
            const submitted = await editor.onSubmit(request, resolution.target);
            if (submitted) canvas.send({ type: "selection.cleared" });
            return submitted;
          }}
          onClearSelection={() => canvas.send({ type: "selection.cleared" })}
        />
      </div>
    </>
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
