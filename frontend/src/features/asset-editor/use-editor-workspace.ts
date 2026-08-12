import { useEffect, useMemo, useState } from "react";

import { i18n } from "@/i18n";
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
  const t = (key: string, options?: Record<string, unknown>) =>
    i18n.getFixedT(null, "editor")(key as never, options);
  const { asset, projectName } = data;
  const session = useEditorSession({
    target: { projectId: asset.projectId, assetId: asset.id },
    initialRecord: data.record,
  });
  const { snapshot } = session;
  const animationMutation = useGenerateAnimationMutation();
  const { data: generationRuns = [] } = useGenerationRunsQuery(asset.projectId);
  const [animationTask, setAnimationTask] =
    useState<EditorGenerationTask | null>(null);
  const [promptTask, setPromptTask] = useState<EditorGenerationTask | null>(
    null,
  );
  const [notice, setNotice] = useState<string | null>(null);
  const { schedule: scheduleNoticeReset } = useTimeout();
  const { schedule: schedulePromptTaskReset } = useTimeout();

  useEffect(() => {
    setNotice(null);
    setAnimationTask(null);
    setPromptTask(null);
  }, [asset.id, asset.projectId]);

  const generationTasks = useMemo<EditorGenerationTask[]>(
    () => [
      ...generationRuns.flatMap((run) =>
        run.status === "pending" || run.status === "processing"
          ? [
              {
                id: run.id,
                name: run.name,
                prompt: run.prompt,
                status: run.status === "pending" ? "queued" : "processing",
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
    labels: {
      saving: t("savingChanges"),
      sendingPrompt: t("sendingPrompt"),
      generatingAnimation: t("generatingAnimationStatus"),
      unsavedChanges: t("unsavedChanges"),
      allChangesSaved: t("allChangesSaved"),
    },
  });

  const save = async () => {
    if (!snapshot.dirty) return;

    const result = await session.save();
    if (result.status === "saved") reportAction(t("savedJustNow"));
    if (result.status === "failed") reportAction(t("saveFailed"));
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
      name: request.label,
      prompt: request.prompt,
      status: "processing",
    });

    try {
      const result = await animationMutation.mutateAsync({
        ...request,
        projectId: asset.projectId,
        assetId: asset.id,
        assetKind,
        prototype: sprite.prototype,
      });
      session.dispatch({
        type: "sprite.animation.generated",
        animation: result.animation,
      });
      reportAction(t("animationGenerated", { name: request.label }));
    } catch {
      reportAction(t("animationGenerationFailed"));
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

      reportAction(t("promptSent"));
      schedulePromptTaskReset(() => {
        setPromptTask((current) => (current?.id === taskId ? null : current));
      }, 1800);
    } catch (error) {
      setPromptTask((current) => (current?.id === taskId ? null : current));
      reportAction(t("promptSubmissionFailed"));
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
      prompt: snapshot.record.prompt,
      history: asset.history,
      isSubmitting: promptTask !== null,
      onPromptChange: (value) =>
        session.dispatch({ type: "prompt.set", value }),
      onSubmit: submitInspectorPrompt,
    },
  };
}
