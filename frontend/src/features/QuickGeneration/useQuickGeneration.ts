import { useEffect, useRef, useState } from "react";

import {
  useDeleteQuickAssetMutation,
  useGenerateQuickAssetMutation,
  useQuickAssetsQuery,
} from "@/api/quick-generation";
import {
  createQuickGenerationDraft,
  quickGenerationSizes,
  toGenerateQuickAssetInput,
  type QuickGenerationAsset,
} from "@/domain/quick-generation";

export function useQuickGeneration() {
  const assetsQuery = useQuickAssetsQuery();
  const generateMutation = useGenerateQuickAssetMutation();
  const deleteMutation = useDeleteQuickAssetMutation();
  const [currentAssetId, setCurrentAssetId] = useState<
    string | null | undefined
  >(undefined);
  const [draft, setDraft] = useState(() =>
    createQuickGenerationDraft<string>(),
  );
  const referencePreviews = useRef<Record<string, string>>({});
  const ownedReferenceUrls = useRef(new Set<string>());
  const assets = assetsQuery.data ?? [];
  const currentAsset =
    typeof currentAssetId === "string"
      ? assets.find((asset) => asset.id === currentAssetId)
      : undefined;
  const isMutating = generateMutation.isPending || deleteMutation.isPending;
  const actionError = generateMutation.error ?? deleteMutation.error;

  useEffect(() => {
    if (assetsQuery.isPending || currentAssetId !== undefined) return;
    const firstAsset = assets[0];
    setCurrentAssetId(firstAsset?.id ?? null);
    setDraft(createQuickGenerationDraft(firstAsset));
  }, [assets, assetsQuery.isPending, currentAssetId]);

  useEffect(
    () => () => {
      for (const url of ownedReferenceUrls.current) {
        URL.revokeObjectURL(url);
      }
      ownedReferenceUrls.current.clear();
    },
    [],
  );

  function selectAsset(asset: QuickGenerationAsset) {
    releaseUncommittedReference();
    setCurrentAssetId(asset.id);
    setDraft(
      createQuickGenerationDraft(
        asset,
        referencePreviews.current[asset.id] ?? "",
      ),
    );
  }

  function newAsset() {
    releaseUncommittedReference();
    setCurrentAssetId(null);
    setDraft(createQuickGenerationDraft());
  }

  function generate() {
    const request = toGenerateQuickAssetInput(draft);
    if (!request) return;
    const submittedReferenceImage = draft.reference ?? "";

    generateMutation.mutate(request, {
      onSuccess: (asset) => {
        commitReferencePreview(asset.id, submittedReferenceImage);
        setCurrentAssetId(asset.id);
        setDraft(createQuickGenerationDraft(asset, submittedReferenceImage));
      },
    });
  }

  function deleteCurrentAsset() {
    if (!currentAsset) return;
    const deletedAssetId = currentAsset.id;
    const remaining = assets.filter((asset) => asset.id !== deletedAssetId);

    deleteMutation.mutate(deletedAssetId, {
      onSuccess: () => {
        releaseUncommittedReference();
        releaseReferencePreview(deletedAssetId);
        const nextAsset = remaining[0];
        setCurrentAssetId(nextAsset?.id ?? null);
        setDraft(
          createQuickGenerationDraft(
            nextAsset,
            nextAsset ? (referencePreviews.current[nextAsset.id] ?? "") : "",
          ),
        );
      },
    });
  }

  function chooseReference(file: File | undefined) {
    if (!file || !file.type.startsWith("image/")) return;
    releaseUncommittedReference();
    const previewUrl = URL.createObjectURL(file);
    ownedReferenceUrls.current.add(previewUrl);
    setDraft((current) => ({
      ...current,
      reference: previewUrl,
      referenceFileName: file.name,
    }));
  }

  function clearReference() {
    releaseUncommittedReference();
    setDraft((current) => ({
      ...current,
      reference: "",
      referenceFileName: undefined,
    }));
  }

  function releaseUncommittedReference() {
    const committedReference =
      typeof currentAssetId === "string"
        ? referencePreviews.current[currentAssetId]
        : undefined;
    if (draft.reference && draft.reference !== committedReference) {
      revokeOwnedUrl(draft.reference);
    }
  }

  function commitReferencePreview(assetId: string, previewUrl: string) {
    const previousPreview = referencePreviews.current[assetId];
    if (previousPreview && previousPreview !== previewUrl) {
      revokeOwnedUrl(previousPreview);
    }
    if (previewUrl) {
      referencePreviews.current[assetId] = previewUrl;
    } else {
      delete referencePreviews.current[assetId];
    }
  }

  function releaseReferencePreview(assetId: string) {
    const previewUrl = referencePreviews.current[assetId];
    if (previewUrl) {
      revokeOwnedUrl(previewUrl);
      delete referencePreviews.current[assetId];
    }
  }

  function revokeOwnedUrl(url: string) {
    if (!ownedReferenceUrls.current.has(url)) return;
    URL.revokeObjectURL(url);
    ownedReferenceUrls.current.delete(url);
  }

  return {
    actionError,
    assets,
    chooseReference,
    clearReference,
    currentAsset,
    currentAssetId: currentAssetId ?? null,
    deleteCurrentAsset,
    description: draft.prompt,
    generate,
    isDeleting: deleteMutation.isPending,
    isGenerating: generateMutation.isPending,
    isLoading: assetsQuery.isPending,
    isMutating,
    loadError: assetsQuery.error,
    newAsset,
    quickGenerationSizes,
    referenceImage: draft.reference ?? "",
    reload: assetsQuery.refetch,
    selectAsset,
    setDescription: (prompt: string) =>
      setDraft((current) => ({ ...current, prompt })),
    setSize: (size: string) => setDraft((current) => ({ ...current, size })),
    size: draft.size,
  };
}
