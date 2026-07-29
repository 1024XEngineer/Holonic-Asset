import { useEffect, useState } from "react";

import { useGenerateAnimationMutation, useGenerationRunsQuery } from "@/api";
import type {
  EditorCanvasPosition,
  EditorWorkspaceData,
  GenerateAnimationRequest,
} from "@/model";
import { useTimeout } from "@/hooks/use-timeout";

import { useAssetEditorSession } from "./state/session";
import { AudioEditorMode } from "./EditorModes/audio-editor-mode";
import { CharacterEditorMode } from "./EditorModes/character-editor-mode";
import { SceneryEditorMode } from "./EditorModes/scenery-editor-mode";
import { TilesetEditorMode } from "./EditorModes/tileset-editor-mode";
import { UiEditorMode } from "./EditorModes/ui-editor-mode";
import { EditorHeader } from "./Header/editor-header";
import type { EditorGenerationTask } from "./Header/editor-header";

export function EditorWorkspace({
  data,
  onBack,
}: {
  data: EditorWorkspaceData;
  onBack: () => void;
}) {
  const { asset, projectName } = data;
  const session = useAssetEditorSession({
    target: {
      projectId: asset.projectId,
      assetId: asset.id,
    },
    initialRecord: data.record,
  });
  const { snapshot } = session;
  const generateAnimationMutation = useGenerateAnimationMutation();
  const { data: generationRuns = [] } = useGenerationRunsQuery(asset.projectId);
  const [animationGenerationTask, setAnimationGenerationTask] =
    useState<EditorGenerationTask | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const { schedule: scheduleNoticeReset } = useTimeout();

  useEffect(() => {
    setNotice(null);
    setAnimationGenerationTask(null);
  }, [asset.projectId, asset.id]);

  const generationTasks: EditorGenerationTask[] = [
    ...generationRuns.flatMap((run) =>
      run.status === "queued" || run.status === "processing"
        ? [
            {
              id: run.id,
              name: run.name,
              prompt: run.prompt,
              status: run.status,
            },
          ]
        : [],
    ),
    ...(animationGenerationTask ? [animationGenerationTask] : []),
  ];

  const reportAction = (message: string) => {
    setNotice(message);
    scheduleNoticeReset(() => setNotice(null), 2200);
  };
  const status =
    snapshot.saveState.phase === "saving"
      ? "Saving changes"
      : generateAnimationMutation.isPending
        ? "Generating animation"
        : (notice ??
          (snapshot.saveState.phase === "failed"
            ? snapshot.saveState.message
            : snapshot.dirty
              ? "Unsaved changes"
              : "All changes saved"));
  const undo = () => {
    session.dispatch({ type: "history.undo" });
    reportAction("Last edit reverted");
  };
  const redo = () => {
    session.dispatch({ type: "history.redo" });
    reportAction("Edit restored");
  };
  const save = async () => {
    const result = await session.save();
    if (result.status === "saved") reportAction("Saved just now");
    if (result.status === "failed") reportAction("Save failed");
  };

  const generateAnimation = async (request: GenerateAnimationRequest) => {
    if (snapshot.record.mode !== "character") return;

    const taskId = `animation-${crypto.randomUUID()}`;
    setAnimationGenerationTask({
      id: taskId,
      name: request.label,
      prompt: request.prompt,
      status: "processing",
    });

    try {
      const result = await generateAnimationMutation.mutateAsync({
        ...request,
        projectId: asset.projectId,
        assetId: asset.id,
        prototype: snapshot.record.character.prototype,
      });
      session.dispatch({
        type: "character.animation.generated",
        animation: result.animation,
      });
      reportAction(`${request.label} generated. Save to keep it.`);
    } catch {
      reportAction("Animation generation failed");
    } finally {
      setAnimationGenerationTask((current) =>
        current?.id === taskId ? null : current,
      );
    }
  };

  const renderHeader = (_selection: string) => (
    <EditorHeader
      assetName={asset.name}
      version={asset.version}
      projectName={projectName}
      onBack={onBack}
      status={status}
      canUndo={snapshot.canUndo}
      canRedo={snapshot.canRedo}
      isSaving={snapshot.saveState.phase === "saving"}
      generationTasks={generationTasks}
      onUndo={undo}
      onRedo={redo}
      onSave={() => void save()}
    />
  );
  const modeProps = {
    prompt: snapshot.record.prompt,
    history: asset.history,
    onAction: reportAction,
    onPromptChange: (value: string) =>
      session.dispatch({ type: "prompt.set", value }),
    renderHeader,
  };

  const editorMode = (() => {
    switch (snapshot.record.mode) {
      case "character":
        return (
          <CharacterEditorMode
            {...modeProps}
            characterPrototype={snapshot.record.character.prototype}
            characterAnimations={snapshot.record.character.animations ?? []}
            characterNodePositions={snapshot.record.character.nodePositions}
            onCharacterPositionChange={(
              nodeId: string,
              position: EditorCanvasPosition,
            ) =>
              session.dispatch({
                type: "character.node-position.set",
                nodeId,
                position,
              })
            }
            onCharacterAnimationGenerate={(request) =>
              void generateAnimation(request)
            }
            onCharacterAnimationRename={(animationId, label) =>
              session.dispatch({
                type: "character.animation.rename",
                animationId,
                label,
              })
            }
            onCharacterAnimationDelete={(animationId) =>
              session.dispatch({
                type: "character.animation.delete",
                animationId,
              })
            }
            isGeneratingAnimation={generateAnimationMutation.isPending}
          />
        );
      case "scenery":
        return (
          <SceneryEditorMode
            {...modeProps}
            layers={snapshot.record.scenery.layers}
          />
        );
      case "tileset":
        return (
          <TilesetEditorMode {...modeProps} tileset={snapshot.record.tileset} />
        );
      case "ui":
        return <UiEditorMode {...modeProps} ui={snapshot.record.ui} />;
      case "audio":
        return <AudioEditorMode {...modeProps} />;
    }
  })();

  return (
    <div className="asset-workspace-shell flex h-screen min-h-0 w-screen flex-col overflow-hidden bg-[#f7f5f0] text-[#2d2923] selection:bg-[#d99096] selection:text-[#2d2923]">
      {editorMode}
    </div>
  );
}
