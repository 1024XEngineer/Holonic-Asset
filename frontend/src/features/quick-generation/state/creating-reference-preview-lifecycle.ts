export type CreatingReferencePreviewAdapter = {
  createPreviewUrl: (file: File) => string;
  revokePreviewUrl: (url: string) => void;
};

export type CreatingReferencePreviewLifecycle = {
  create: (file: File) => string;
  previewForAsset: (assetId: string) => string | undefined;
  releaseUncommitted: (
    draftCreatingReference: string | undefined,
    currentAssetId: string | null,
  ) => void;
  retainForSubmission: (url: string) => void;
  commit: (assetId: string, url: string) => void;
  settleSubmission: (
    url: string,
    keep: boolean,
    currentDraftCreatingReference: string | undefined,
  ) => void;
  releaseAsset: (assetId: string) => void;
  dispose: () => void;
};

export const browserCreatingReferencePreviewAdapter: CreatingReferencePreviewAdapter =
  {
    createPreviewUrl: (file) => URL.createObjectURL(file),
    revokePreviewUrl: (url) => URL.revokeObjectURL(url),
  };

export function createCreatingReferencePreviewLifecycle(
  adapter: CreatingReferencePreviewAdapter = browserCreatingReferencePreviewAdapter,
): CreatingReferencePreviewLifecycle {
  const previewsByAsset = new Map<string, string>();
  const ownedUrls = new Set<string>();
  const retainedUrls = new Set<string>();
  const deferredRevocations = new Set<string>();

  function forceRevoke(url: string) {
    if (!ownedUrls.has(url)) return;
    adapter.revokePreviewUrl(url);
    ownedUrls.delete(url);
    deferredRevocations.delete(url);
  }

  function revoke(url: string) {
    if (!ownedUrls.has(url)) return;
    if (retainedUrls.has(url)) {
      deferredRevocations.add(url);
      return;
    }
    forceRevoke(url);
  }

  return {
    create: (file) => {
      const url = adapter.createPreviewUrl(file);
      ownedUrls.add(url);
      return url;
    },
    previewForAsset: (assetId) => previewsByAsset.get(assetId),
    releaseUncommitted: (draftCreatingReference, currentAssetId) => {
      const committedCreatingReference = currentAssetId
        ? previewsByAsset.get(currentAssetId)
        : undefined;
      if (
        draftCreatingReference &&
        draftCreatingReference !== committedCreatingReference
      ) {
        revoke(draftCreatingReference);
      }
    },
    retainForSubmission: (url) => {
      if (ownedUrls.has(url)) retainedUrls.add(url);
    },
    commit: (assetId, url) => {
      const previousPreview = previewsByAsset.get(assetId);
      if (previousPreview && previousPreview !== url) revoke(previousPreview);
      if (url) previewsByAsset.set(assetId, url);
      else previewsByAsset.delete(assetId);
    },
    settleSubmission: (url, keep, currentDraftCreatingReference) => {
      if (!url || !ownedUrls.has(url)) return;
      retainedUrls.delete(url);
      if (keep) {
        deferredRevocations.delete(url);
        return;
      }
      if (
        deferredRevocations.has(url) ||
        (currentDraftCreatingReference !== url &&
          ![...previewsByAsset.values()].includes(url))
      ) {
        forceRevoke(url);
      }
    },
    releaseAsset: (assetId) => {
      const previewUrl = previewsByAsset.get(assetId);
      if (!previewUrl) return;
      revoke(previewUrl);
      previewsByAsset.delete(assetId);
    },
    dispose: () => {
      for (const url of ownedUrls) adapter.revokePreviewUrl(url);
      ownedUrls.clear();
      retainedUrls.clear();
      deferredRevocations.clear();
      previewsByAsset.clear();
    },
  };
}
