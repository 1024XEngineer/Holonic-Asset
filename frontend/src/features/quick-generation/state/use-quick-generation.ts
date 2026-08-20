import { useEffect, useRef, useSyncExternalStore } from "react";

import {
  useDeleteQuickAssetMutation,
  useGenerateQuickAssetMutation,
  useQuickAssetsQuery,
} from "@/model";
import {
  createQuickGenerationSession,
  type QuickGenerationSession,
} from "./quick-generation-session";

export function useQuickGeneration() {
  const assetsQuery = useQuickAssetsQuery();
  const generateMutation = useGenerateQuickAssetMutation();
  const deleteMutation = useDeleteQuickAssetMutation();
  const sessionRef = useRef<QuickGenerationSession | null>(null);
  if (sessionRef.current === null) {
    sessionRef.current = createQuickGenerationSession();
  }
  const session = sessionRef.current;
  const sessionSnapshot = useSyncExternalStore(
    session.subscribe,
    session.getSnapshot,
    session.getSnapshot,
  );
  const assets = assetsQuery.data ?? [];
  const currentAsset =
    sessionSnapshot.currentAssetId === null
      ? undefined
      : assets.find((asset) => asset.id === sessionSnapshot.currentAssetId);
  const status = {
    actionError: generateMutation.error ?? deleteMutation.error,
    isDeleting: deleteMutation.isPending,
    isGenerating: generateMutation.isPending,
    isLoading: assetsQuery.isPending,
    isMutating: generateMutation.isPending || deleteMutation.isPending,
    loadError: assetsQuery.error,
  };

  useEffect(() => {
    if (assetsQuery.isSuccess) session.synchronize(assetsQuery.data);
  }, [assetsQuery.data, assetsQuery.isSuccess, session]);

  useEffect(() => () => session.dispose(), [session]);

  function generate() {
    const submission = session.prepareGeneration();
    if (!submission) return;
    generateMutation.mutate(submission.input, {
      onSuccess: submission.complete,
      onError: submission.fail,
    });
  }

  function deleteCurrentAsset() {
    const deletion = session.prepareDeletion(assets);
    if (!deletion) return;
    deleteMutation.mutate(deletion.assetId, {
      onSuccess: deletion.complete,
    });
  }

  return {
    model: {
      assets,
      currentAsset,
      currentAssetId: sessionSnapshot.currentAssetId,
      draft: sessionSnapshot.draft,
      status,
    },
    actions: {
      chooseReference: session.chooseReference,
      clearReference: session.clearReference,
      deleteCurrentAsset,
      generate,
      reload: assetsQuery.refetch,
      selectAsset: session.selectAsset,
      startNewAsset: session.startNewAsset,
      updateDraft: session.updateDraft,
    },
  };
}

export type QuickGenerationController = ReturnType<typeof useQuickGeneration>;
