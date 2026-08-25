import { useEffect, useMemo, useState } from "react";

import {
  type GenerationTaskListItem,
  useGenerationEditFlow,
} from "@/features/generation";
import { useTimeout } from "@/hooks/use-timeout";
import type {
  AssetRecord,
  AssetWorkspaceData,
  CreateGenerationRequest,
} from "@/model";

import { getEditorStatus } from "./editor-status";
import { useEditorSession } from "./state";

type EditorGenerationWorkspaceInput<Content> = {
  data: AssetWorkspaceData;
  onBack: () => void;
  toCandidateRecord: (record: AssetRecord, content: Content) => AssetRecord;
  additionalTasks?: GenerationTaskListItem[];
  isAdditionalGenerationPending?: boolean;
};

export function useEditorGenerationWorkspace<Content>({
  data,
  onBack,
  toCandidateRecord,
  additionalTasks = [],
  isAdditionalGenerationPending = false,
}: EditorGenerationWorkspaceInput<Content>) {
  const { asset, projectName } = data;
  const session = useEditorSession({
    target: {
      projectId: asset.projectId,
      assetId: asset.id,
      version: asset.version,
    },
    initialRecord: data.record,
  });
  const { snapshot, syncExternalRecord } = session;
  const generation = useGenerationEditFlow<Content>({
    projectId: asset.projectId,
    assetId: asset.id,
    assetKind: asset.kind,
    assetName: asset.name,
  });
  const [prompt, setPrompt] = useState("");
  const [notice, setNotice] = useState<string | null>(null);
  const { schedule: scheduleNoticeReset } = useTimeout();

  useEffect(() => {
    setNotice(null);
    setPrompt("");
  }, [asset.id, asset.projectId]);

  useEffect(() => {
    syncExternalRecord(data.record);
  }, [data.record, syncExternalRecord]);

  const reportAction = (message: string) => {
    setNotice(message);
    scheduleNoticeReset(() => setNotice(null), 2400);
  };

  const generationTasks = useMemo<GenerationTaskListItem[]>(() => {
    const runTasks = generation.runs.flatMap((run) => {
      if (
        run.status !== "pending" &&
        run.status !== "processing" &&
        run.status !== "failed"
      ) {
        return [];
      }

      return [
        {
          id: run.id,
          name: run.name,
          prompt: run.prompt,
          status: run.status,
          projectId: run.projectId,
          kind: run.kind,
          ...(run.error ? { error: run.error } : {}),
        } satisfies GenerationTaskListItem,
      ];
    });
    const submittedTask = generation.submittedTask
      ? [{ ...generation.submittedTask, status: "processing" as const }]
      : [];

    return [...runTasks, ...additionalTasks, ...submittedTask];
  }, [additionalTasks, generation.runs, generation.submittedTask]);

  const candidateRecord = useMemo(() => {
    if (generation.candidateContent === undefined) return null;
    try {
      return toCandidateRecord(snapshot.record, generation.candidateContent);
    } catch {
      return null;
    }
  }, [generation.candidateContent, snapshot.record, toCandidateRecord]);

  const save = async () => {
    if (!snapshot.dirty) return;
    const result = await session.save();
    if (result.status === "saved") reportAction("Saved just now");
    if (result.status === "failed") reportAction("Save failed");
  };

  const submit = async ({
    request,
    prompt: submittedPrompt,
  }: {
    request: CreateGenerationRequest;
    prompt: string;
  }) => {
    try {
      const submitted = await generation.submit({
        request,
        prompt: submittedPrompt,
      });
      if (submitted) reportAction("Prompt sent");
    } catch (error) {
      reportAction("Prompt submission failed");
      throw error;
    }
  };

  const resolveReview = async (applied: boolean) => {
    try {
      const resolved = await generation.resolveReview(applied);
      if (!resolved) return;
      if (applied && candidateRecord) {
        session.dispatch({
          type: "record.candidate.apply",
          record: candidateRecord,
        });
      }
      reportAction(applied ? "Generation applied" : "Generation denied");
    } catch {
      reportAction("Unable to consume generation result");
    }
  };

  return {
    header: {
      assetKind: asset.kind,
      assetName: asset.name,
      version: asset.version,
      projectName,
      status: getEditorStatus({
        saveState: snapshot.saveState,
        isPromptSubmitting: generation.isSubmitting,
        isGeneratingAnimation: isAdditionalGenerationPending,
        notice,
        isDirty: snapshot.dirty,
      }),
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
    snapshot,
    session,
    prompt,
    setPrompt,
    isPromptSubmitting: generation.isSubmitting,
    candidateContent: generation.candidateContent,
    candidateRecord,
    candidateKind: generation.candidateKind,
    candidateAnimationId: generation.candidateAnimationId,
    reviewRun: generation.reviewRun,
    isResolvingReview: generation.isResolvingReview,
    reportAction,
    submit,
    resolveReview,
  };
}
