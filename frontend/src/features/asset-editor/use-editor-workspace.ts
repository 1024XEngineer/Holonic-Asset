import { useMemo, useState } from "react";

import {
  buildSpriteGenerationRequest,
  type GenerationTaskListItem,
} from "@/features/generation";
import {
  toCoreSpriteCandidateRecord,
  useGenerateAnimationMutation,
  type AssetWorkspaceData,
  type CoreSpriteAssetContentPatch,
  type GenerateAnimationRequest,
  type GenerationTaskType,
} from "@/model";

import type { AnimatedSpriteCanvasReview } from "./Canvas/AnimatedSpriteCanvas";
import type { SpriteEditorModeProps } from "./EditorModes/sprite-editor-mode.types";
import type { InspectorSubmitRequest } from "./Inspector/inspector.types";
import { useEditorGenerationWorkspace } from "./use-editor-generation-workspace";

export function useEditorWorkspace({
  data,
  onBack,
}: {
  data: AssetWorkspaceData;
  onBack: () => void;
}): SpriteEditorModeProps | null {
  const { asset } = data;
  const animationMutation = useGenerateAnimationMutation();
  const [animationTask, setAnimationTask] =
    useState<GenerationTaskListItem | null>(null);
  const additionalTasks = useMemo(
    () => (animationTask ? [animationTask] : []),
    [animationTask],
  );
  const flow = useEditorGenerationWorkspace<CoreSpriteAssetContentPatch>({
    data,
    onBack,
    additionalTasks,
    isAdditionalGenerationPending: animationMutation.isPending,
    toCandidateRecord: (record, content) =>
      toCoreSpriteCandidateRecord(record, asset.perspective, content),
  });
  const { snapshot, session, candidateRecord, reportAction } = flow;

  if (
    snapshot.record.mode !== "character" &&
    snapshot.record.mode !== "object"
  ) {
    return null;
  }

  const reviewKind = flow.candidateKind;
  const displayRecord =
    reviewKind === "generate_animation" && candidateRecord
      ? candidateRecord
      : snapshot.record;
  const sprite = getSpriteRecordData(displayRecord);
  const generationReview = buildGenerationReview({
    taskKind: reviewKind,
    currentRecord: snapshot.record,
    candidateRecord,
    animationId: flow.candidateAnimationId,
    isResolving: flow.isResolvingReview,
  });
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
    const assetId = Number(asset.id);
    await flow.submit({
      prompt: request.prompt,
      request: buildSpriteGenerationRequest(assetKind, assetId, request),
    });
  };

  return {
    header: flow.header,
    ...(flow.reviewRun && generationReview
      ? {
          generationReview: {
            ...generationReview,
            onApply: () => void flow.resolveReview(true),
            onDeny: () => void flow.resolveReview(false),
          },
        }
      : {}),
    sprite: {
      perspective: asset.perspective,
      prototype: sprite.prototype,
      animations: sprite.animations ?? [],
      nodePositions: {},
      onPositionChange: () => undefined,
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
      prompt: flow.prompt,
      history: asset.history,
      isSubmitting: flow.isPromptSubmitting,
      onPromptChange: flow.setPrompt,
      onSubmit: submitInspectorPrompt,
    },
  };
}

function getSpriteRecordData(record: AssetWorkspaceData["record"]) {
  if (record.mode === "character") return record.character;
  if (record.mode === "object") return record.object;
  throw new Error("Sprite editor requires a Character or Object asset.");
}

function buildGenerationReview({
  taskKind,
  currentRecord,
  candidateRecord,
  animationId,
  isResolving,
}: {
  taskKind: GenerationTaskType | undefined;
  currentRecord: AssetWorkspaceData["record"];
  candidateRecord: AssetWorkspaceData["record"] | null;
  animationId?: number;
  isResolving: boolean;
}): AnimatedSpriteCanvasReview | undefined {
  if (!taskKind || !candidateRecord) return undefined;
  const current = getSpriteRecordData(currentRecord);
  const candidate = getSpriteRecordData(candidateRecord);

  if (taskKind === "generate_animation") {
    const animation = findCandidateAnimation(
      current.animations ?? [],
      candidate.animations ?? [],
      animationId,
    );
    return animation
      ? {
          kind: "new-animation",
          nodeId: animation.id,
          isResolving,
        }
      : undefined;
  }

  if (
    taskKind === "edit_character_prototype" ||
    taskKind === "edit_object_prototype"
  ) {
    return {
      kind: "comparison",
      nodeId: "prototype",
      candidatePrototype: candidate.prototype,
      isResolving,
    };
  }

  if (taskKind === "edit_animation" || taskKind === "edit_frames") {
    const animation = findCandidateAnimation(
      current.animations ?? [],
      candidate.animations ?? [],
      animationId,
    );
    return animation
      ? {
          kind: "comparison",
          nodeId: animation.id,
          candidateAnimation: animation,
          isResolving,
        }
      : undefined;
  }
  return undefined;
}

function findCandidateAnimation(
  current: ReturnType<typeof getSpriteRecordData>["animations"],
  candidate: ReturnType<typeof getSpriteRecordData>["animations"],
  animationId?: number,
) {
  const targetId = animationId === undefined ? undefined : String(animationId);
  return (
    candidate?.find((animation) => animation.id === targetId) ??
    candidate?.find(
      (animation) =>
        !current?.some(
          (currentAnimation) => currentAnimation.id === animation.id,
        ),
    )
  );
}
