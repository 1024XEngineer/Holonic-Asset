import { useEffect, useRef, useState } from "react";

import {
  useDeleteQuickAssetMutation,
  useGenerateQuickAssetMutation,
  useQuickAssetsQuery,
} from "@/api/quick-generation";
import type { QuickGenerationAsset } from "@/domain/quick-generation";
import { quickGenerationSizes } from "./QuickGeneration.constants";

export function useQuickGeneration() {
  const assetsQuery = useQuickAssetsQuery();
  const generateMutation = useGenerateQuickAssetMutation();
  const deleteMutation = useDeleteQuickAssetMutation();
  const [currentAssetId, setCurrentAssetId] = useState<
    string | null | undefined
  >(undefined);
  const [description, setDescription] = useState("");
  const [referenceImage, setReferenceImage] = useState("");
  const [referenceFileName, setReferenceFileName] = useState<
    string | undefined
  >();
  const [size, setSize] = useState(quickGenerationSizes[1]);
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
    if (firstAsset) {
      setSize(firstAsset.size);
      setReferenceFileName(firstAsset.referenceFileName);
    }
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
    setDescription("");
    setReferenceImage(referencePreviews.current[asset.id] ?? "");
    setReferenceFileName(asset.referenceFileName);
    setSize(asset.size);
  }

  function newAsset() {
    releaseUncommittedReference();
    setCurrentAssetId(null);
    setDescription("");
    setReferenceImage("");
    setReferenceFileName(undefined);
    setSize(quickGenerationSizes[1]);
  }

  function generate() {
    if (!description.trim()) return;
    const previousAssetId = currentAsset?.id;
    const submittedReferenceImage = referenceImage;

    generateMutation.mutate(
      {
        assetId: previousAssetId,
        prompt: description,
        size,
        referenceFileName,
      },
      {
        onSuccess: (asset) => {
          commitReferencePreview(asset.id, submittedReferenceImage);
          setCurrentAssetId(asset.id);
          setDescription("");
        },
      },
    );
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
        setReferenceImage(
          nextAsset ? (referencePreviews.current[nextAsset.id] ?? "") : "",
        );
        setReferenceFileName(nextAsset?.referenceFileName);
        setSize(nextAsset?.size ?? quickGenerationSizes[1]);
        setDescription("");
      },
    });
  }

  function chooseReference(file: File | undefined) {
    if (!file || !file.type.startsWith("image/")) return;
    releaseUncommittedReference();
    const previewUrl = URL.createObjectURL(file);
    ownedReferenceUrls.current.add(previewUrl);
    setReferenceImage(previewUrl);
    setReferenceFileName(file.name);
  }

  function clearReference() {
    releaseUncommittedReference();
    setReferenceImage("");
    setReferenceFileName(undefined);
  }

  function releaseUncommittedReference() {
    const committedReference =
      typeof currentAssetId === "string"
        ? referencePreviews.current[currentAssetId]
        : undefined;
    if (referenceImage && referenceImage !== committedReference) {
      revokeOwnedUrl(referenceImage);
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
    setReferenceImage(previewUrl);
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
    description,
    generate,
    isDeleting: deleteMutation.isPending,
    isGenerating: generateMutation.isPending,
    isLoading: assetsQuery.isPending,
    isMutating,
    loadError: assetsQuery.error,
    newAsset,
    quickGenerationSizes,
    referenceImage,
    reload: assetsQuery.refetch,
    selectAsset,
    setDescription,
    setSize,
    size,
  };
}
