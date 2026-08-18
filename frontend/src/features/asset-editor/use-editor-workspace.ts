import { useEffect, useMemo, useState } from "react";

import { useTimeout } from "@/hooks/use-timeout";
import {
  coreGenerationApi,
  rememberGenerationRunMetadata,
  toCoreSpriteCandidateRecord,
  useGenerateAnimationMutation,
  useGenerationCandidateQuery,
  useGenerationRunsQuery,
  useResolveGenerationApplicationMutation,
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
    target: {
      projectId: asset.projectId,
      assetId: asset.id,
      version: asset.version,
    },
    initialRecord: data.record,
  });
  const { snapshot } = session;
  const animationMutation = useGenerateAnimationMutation();
  const applicationMutation = useResolveGenerationApplicationMutation();
  const { data: generationRuns = [] } = useGenerationRunsQuery(
    asset.projectId,
    asset.id,
  );
  const awaitingRuns = useMemo(
    () => generationRuns.filter((run) => run.status === "awaiting_application"),
    [generationRuns],
  );
  const reviewRun = awaitingRuns[0];
  const candidateQuery = useGenerationCandidateQuery(reviewRun?.id);
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

  const reportAction = (message: string) => {
    setNotice(message);
    scheduleNoticeReset(() => setNotice(null), 2400);
  };
  const resolveApplication = async (runId: string, applied: boolean) => {
    try {
      await applicationMutation.mutateAsync({
        projectId: asset.projectId,
        assetId: asset.id,
        runId,
        applied,
      });
      reportAction(applied ? "Generation applied" : "Generation denied");
    } catch {
      reportAction("Unable to consume generation result");
    }
  };

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

  const candidateRecord = useMemo(() => {
    const content = candidateQuery.data?.result?.content;
    if (content === undefined) return null;
    try {
      return toCoreSpriteCandidateRecord(
        snapshot.record,
        asset.perspective,
        content,
      );
    } catch {
      return null;
    }
  }, [asset.perspective, candidateQuery.data, snapshot.record]);

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

  const displayRecord = candidateRecord ?? snapshot.record;
  const sprite = getSpriteRecordData(displayRecord);
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
        const created = await coreGenerationApi.create(
          projectId,
          buildInspectorGenerationRequest(assetKind, assetId, request),
        );
        rememberGenerationRunMetadata(
          asset.projectId,
          created.generationRunId,
          {
            kind: assetKind,
            name: `Edit ${asset.name}`,
            prompt,
            assetId: asset.id,
          },
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
    ...(reviewRun
      ? {
          generationReview: {
            name: reviewRun.name,
            prompt: reviewRun.prompt,
            pendingCount: awaitingRuns.length,
            isLoading: candidateQuery.isPending,
            isUnavailable:
              candidateQuery.isError ||
              (!candidateQuery.isPending && candidateRecord === null),
            isResolving: applicationMutation.isPending,
            onApply: () => void resolveApplication(reviewRun.id, true),
            onDeny: () => void resolveApplication(reviewRun.id, false),
          },
        }
      : {}),
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

function getSpriteRecordData(record: AssetWorkspaceData["record"]) {
  if (record.mode === "character") return record.character;
  if (record.mode === "object") return record.object;
  throw new Error("Sprite editor requires a Character or Object asset.");
}
