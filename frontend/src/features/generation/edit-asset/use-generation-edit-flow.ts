import { useEffect, useState } from "react";

import {
  isGenerationRunActive,
  useEnqueueAssetEditGenerationMutation,
  useGenerationCandidateQuery,
  useGenerationRunsQuery,
  useResolveGenerationApplicationMutation,
  type AssetKind,
  type CreateGenerationRequest,
} from "@/model";

type SubmittedAssetEditTask = {
  id: string;
  name: string;
  prompt: string;
};

export function useGenerationEditFlow<Content>({
  projectId,
  assetId,
  assetKind,
  assetName,
}: {
  projectId: string;
  assetId: string;
  assetKind: AssetKind;
  assetName: string;
}) {
  const applicationMutation = useResolveGenerationApplicationMutation();
  const enqueueMutation = useEnqueueAssetEditGenerationMutation();
  const { data: runs = [] } = useGenerationRunsQuery(projectId, assetId);
  const reviewRun = runs.find((run) => run.status === "awaiting_application");
  const candidateQuery = useGenerationCandidateQuery<Content>(reviewRun?.id);
  const [submittedTask, setSubmittedTask] =
    useState<SubmittedAssetEditTask | null>(null);
  const hasActiveRun = runs.some(isGenerationRunActive);

  useEffect(() => {
    setSubmittedTask(null);
  }, [assetId, projectId]);

  const submit = async ({
    request,
    prompt,
  }: {
    request: CreateGenerationRequest;
    prompt: string;
  }) => {
    if (submittedTask || hasActiveRun) return false;
    const taskId = `prompt-${crypto.randomUUID()}`;
    setSubmittedTask({
      id: taskId,
      name: `Edit ${assetName}`,
      prompt,
    });

    try {
      const persistedProjectId = Number(projectId);
      const persistedAssetId = Number(assetId);
      if (
        !Number.isSafeInteger(persistedProjectId) ||
        !Number.isSafeInteger(persistedAssetId)
      ) {
        return true;
      }
      await enqueueMutation.mutateAsync({
        projectId,
        assetId,
        assetKind,
        assetName,
        prompt,
        request,
      });
      return true;
    } finally {
      setSubmittedTask((current) => (current?.id === taskId ? null : current));
    }
  };

  const resolveReview = async (applied: boolean) => {
    if (!reviewRun) return false;
    await applicationMutation.mutateAsync({
      projectId,
      assetId,
      runId: reviewRun.id,
      applied,
    });
    return true;
  };

  return {
    runs,
    submittedTask,
    isSubmitting: submittedTask !== null || hasActiveRun,
    candidateContent: candidateQuery.data?.result?.content,
    candidateKind: candidateQuery.data?.kind,
    candidateAnimationId: candidateQuery.data?.result?.animation_id,
    reviewRun,
    isResolvingReview: applicationMutation.isPending,
    submit,
    resolveReview,
  };
}
