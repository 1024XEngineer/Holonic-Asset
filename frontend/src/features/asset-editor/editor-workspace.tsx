import { useEffect, useMemo, useState } from "react";

import { useTimeout } from "@/hooks/use-timeout";
import {
  useGenerateAnimationMutation,
  useGenerationRunsQuery,
  type AssetWorkspaceData,
  type GenerateAnimationRequest,
} from "@/model";

import { SpriteEditorMode } from "./EditorModes/sprite-editor-mode";
import type { EditorGenerationTask } from "./Header/editor-header";
import { useEditorSession } from "./state";

export function EditorWorkspace({
  data,
  onBack,
}: {
  data: AssetWorkspaceData;
  onBack: () => void;
}) {
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
  const [notice, setNotice] = useState<string | null>(null);
  const { schedule: scheduleNoticeReset } = useTimeout();

  useEffect(() => {
    setNotice(null);
    setAnimationTask(null);
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
    ],
    [animationTask, generationRuns],
  );

  const reportAction = (message: string) => {
    setNotice(message);
    scheduleNoticeReset(() => setNotice(null), 2400);
  };
  const status =
    snapshot.saveState.phase === "saving"
      ? "Saving changes"
      : animationMutation.isPending
        ? "Generating animation"
        : (notice ??
          (snapshot.saveState.phase === "failed"
            ? snapshot.saveState.message
            : snapshot.dirty
              ? "Unsaved changes"
              : "All changes saved"));

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
      reportAction(`${request.label} generated`);
    } catch {
      reportAction("Animation generation failed");
    } finally {
      setAnimationTask((current) => (current?.id === taskId ? null : current));
    }
  };

  return (
    <div className="flex h-dvh min-h-0 w-full flex-col overflow-hidden bg-muted/30 text-foreground selection:bg-primary/20">
      <SpriteEditorMode
        assetKind={assetKind}
        assetName={asset.name}
        version={asset.version}
        projectName={projectName}
        prototype={sprite.prototype}
        animations={sprite.animations ?? []}
        nodePositions={sprite.nodePositions}
        prompt={snapshot.record.prompt}
        history={asset.history}
        status={status}
        canUndo={snapshot.canUndo}
        canRedo={snapshot.canRedo}
        isDirty={snapshot.dirty}
        isSaving={snapshot.saveState.phase === "saving"}
        isGeneratingAnimation={animationMutation.isPending}
        generationTasks={generationTasks}
        onBack={onBack}
        onUndo={() => session.dispatch({ type: "history.undo" })}
        onRedo={() => session.dispatch({ type: "history.redo" })}
        onSave={() => void save()}
        onPromptChange={(value) =>
          session.dispatch({ type: "prompt.set", value })
        }
        onPositionChange={(nodeId, position) =>
          session.dispatch({
            type: "sprite.node-position.set",
            nodeId,
            position,
          })
        }
        onAnimationGenerate={(request) => void generateAnimation(request)}
        onAnimationRename={(animationId, label) =>
          session.dispatch({
            type: "sprite.animation.rename",
            animationId,
            label,
          })
        }
        onAnimationDelete={(animationId) =>
          session.dispatch({
            type: "sprite.animation.delete",
            animationId,
          })
        }
      />
    </div>
  );
}
