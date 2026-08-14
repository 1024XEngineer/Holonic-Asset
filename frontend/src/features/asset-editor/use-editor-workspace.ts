import { useEffect, useMemo, useState } from "react";

import { useTimeout } from "@/hooks/use-timeout";
import {
  coreGenerationApi,
  useGenerateAnimationMutation,
  useGenerationRunsQuery,
  type AssetWorkspaceData,
  type GenerateAnimationRequest,
} from "@/model";

import type { SpriteEditorModeProps } from "./EditorModes/sprite-editor-mode.types";
import type { EditorGenerationTask } from "./Header/editor-header";
import type { InspectorSubmitRequest } from "./Inspector/inspector.types";
import { buildInspectorGenerationRequest } from "./editor-generation-request";
import { getEditorStatus } from "./editor-status";
import { useEditorSession } from "./state";

export function useEditorWorkspace({
  data,
  onBack,
}: {
  data: AssetWorkspaceData;
  onBack: () => void;
}): SpriteEditorModeProps | null {
  const { asset, projectName } = data;
  const session = useEditorSession({
    target: { projectId: asset.projectId, assetId: asset.id },
    initialRecord: data.record,
  });
  const { snapshot } = session;
  const animationMutation = useGenerateAnimationMutation();
  const { data: generationRuns = [] } = useGenerationRunsQuery(
    asset.projectId,
    asset.id,
  );
  const [animationTask, setAnimationTask] =
    useState<EditorGenerationTask | null>(null);
  const [promptTask, setPromptTask] = useState<EditorGenerationTask | null>(
    null,
  );
  const [inspectorPrompt, setInspectorPrompt] = useState("");
  const [notice, setNotice] = useState<string | null>(null);
  const { schedule: scheduleNoticeReset } = useTimeout();
  const { schedule: schedulePromptTaskReset } = useTimeout();

  useEffect(() => {
    setNotice(null);
    setAnimationTask(null);
    setPromptTask(null);
    setInspectorPrompt("");
  }, [asset.id, asset.projectId]);

  useEffect(() => {
    session.syncExternalRecord(data.record);
  }, [data.record, session.syncExternalRecord]);

  const generationTasks = useMemo<EditorGenerationTask[]>(
    () => [
      ...generationRuns.flatMap((run) =>
        run.status === "pending" ||
        run.status === "processing" ||
        run.status === "failed"
          ? [
              {
                id: run.id,
                name: run.name,
                prompt: run.prompt,
                status:
                  run.status === "pending"
                    ? "queued"
                    : run.status === "failed"
                      ? "failed"
                      : "processing",
                ...(run.error ? { error: run.error } : {}),
              } satisfies EditorGenerationTask,
            ]
          : [],
      ),
      ...(animationTask ? [animationTask] : []),
      ...(promptTask ? [promptTask] : []),
    ],
    [animationTask, generationRuns, promptTask],
  );

  const reportAction = (message: string) => {
    setNotice(message);
    scheduleNoticeReset(() => setNotice(null), 2400);
  };
  const status = getEditorStatus({
    saveState: snapshot.saveState,
    isPromptSubmitting: promptTask !== null,
    isGeneratingAnimation: animationMutation.isPending,
    notice,
    isDirty: snapshot.dirty,
  });

  const save = async () => {
    if (!snapshot.dirty) return;

    const result = await session.save();
    if (result.status === "saved") reportAction("Saved just now");
    if (result.status === "failed") reportAction("Save failed");
  };

  if (
    snapshot.record.mode !== "character" &&
    snapshot.record.mode !== "object"
  ) {
    return null;
  }

  const sprite =
    snapshot.record.mode === "character"
      ? snapshot.record.character
      : snapshot.record.object;
  const assetKind = snapshot.record.mode;

  const generateAnimation = async (request: GenerateAnimationRequest) => {
    const taskId = `animation-${crypto.randomUUID()}`;
    setAnimationTask({
      id: taskId,
      name: request.animationName,
      prompt: request.creativeBrief,
      status: "processing",
    });

    try {
      await animationMutation.mutateAsync({
        ...request,
        projectId: asset.projectId,
        assetId: asset.id,
        assetKind,
      });
      reportAction(`${request.animationName} queued`);
    } catch {
      reportAction("Animation generation failed");
    } finally {
      setAnimationTask((current) => (current?.id === taskId ? null : current));
    }
  };

  const submitInspectorPrompt = async (request: InspectorSubmitRequest) => {
    if (promptTask) return;
    const prompt = request.prompt;

    const taskId = `prompt-${crypto.randomUUID()}`;
    setPromptTask({
      id: taskId,
      name: `Edit ${asset.name}`,
      prompt,
      status: "processing",
    });

    try {
      const projectId = Number(asset.projectId);
      const assetId = Number(asset.id);
      if (Number.isSafeInteger(projectId) && Number.isSafeInteger(assetId)) {
        await coreGenerationApi.create(
          projectId,
          buildInspectorGenerationRequest(assetKind, assetId, request),
        );
      }

      reportAction("Prompt sent");
      schedulePromptTaskReset(() => {
        setPromptTask((current) => (current?.id === taskId ? null : current));
      }, 1800);
    } catch (error) {
      setPromptTask((current) => (current?.id === taskId ? null : current));
      reportAction("Prompt submission failed");
      throw error;
    }
  };

  return {
    header: {
      assetKind,
      assetName: asset.name,
      version: asset.version,
      projectName,
      status,
      canUndo: snapshot.canUndo,
      canRedo: snapshot.canRedo,
      isDirty: snapshot.dirty,
      isSaving: snapshot.saveState.phase === "saving",
      generationTasks,
      onBack,
      onUndo: () => session.dispatch({ type: "history.undo" }),
      onRedo: () => session.dispatch({ type: "history.redo" }),
      onSave: () => void save(),
    },
    sprite: {
      perspective: asset.perspective,
      prototype: sprite.prototype,
      animations: sprite.animations ?? [],
      nodePositions: sprite.nodePositions,
      onPositionChange: (nodeId, position) =>
        session.dispatch({
          type: "sprite.node-position.set",
          nodeId,
          position,
        }),
    },
    tree: {
      isGeneratingAnimation: animationMutation.isPending,
      onAnimationGenerate: (request) => void generateAnimation(request),
      onAnimationRename: (animationId, label) =>
        session.dispatch({
          type: "sprite.animation.rename",
          animationId,
          label,
        }),
      onAnimationDelete: (animationId) =>
        session.dispatch({
          type: "sprite.animation.delete",
          animationId,
        }),
    },
    inspector: {
      prompt: inspectorPrompt,
      history: asset.history,
      isSubmitting: promptTask !== null,
      onPromptChange: (value) => setInspectorPrompt(value),
      onSubmit: submitInspectorPrompt,
    },
  };
}
