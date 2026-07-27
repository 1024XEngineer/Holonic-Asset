import {
  createQuickGenerationDraft,
  toGenerateQuickAssetInput,
  type GenerateQuickAssetInput,
  type QuickGenerationAsset,
  type QuickGenerationDraft,
} from "../domain";

type QuickGenerationDraftPatch = Pick<
  QuickGenerationDraft<string>,
  "prompt" | "size"
>;

export type QuickGenerationSessionSnapshot = {
  currentAssetId: string | null;
  draft: QuickGenerationDraft<string>;
};

export type ReferencePreviewAdapter = {
  createPreviewUrl: (file: File) => string;
  revokePreviewUrl: (url: string) => void;
};

export type QuickGenerationSubmission = {
  input: GenerateQuickAssetInput;
  complete: (asset: QuickGenerationAsset) => void;
  fail: () => void;
};

export type QuickGenerationDeletion = {
  assetId: string;
  complete: () => void;
};

export type QuickGenerationSession = {
  subscribe: (listener: () => void) => () => void;
  getSnapshot: () => QuickGenerationSessionSnapshot;
  synchronize: (assets: QuickGenerationAsset[]) => void;
  selectAsset: (asset: QuickGenerationAsset) => void;
  startNewAsset: () => void;
  updateDraft: (patch: Partial<QuickGenerationDraftPatch>) => void;
  chooseReference: (file: File | undefined) => void;
  clearReference: () => void;
  prepareGeneration: () => QuickGenerationSubmission | undefined;
  prepareDeletion: (
    assets: QuickGenerationAsset[],
  ) => QuickGenerationDeletion | undefined;
  dispose: () => void;
};

export const browserReferencePreviewAdapter: ReferencePreviewAdapter = {
  createPreviewUrl: (file) => URL.createObjectURL(file),
  revokePreviewUrl: (url) => URL.revokeObjectURL(url),
};

export function createQuickGenerationSession(
  referencePreview: ReferencePreviewAdapter = browserReferencePreviewAdapter,
): QuickGenerationSession {
  let initialized = false;
  let snapshot: QuickGenerationSessionSnapshot = {
    currentAssetId: null,
    draft: createQuickGenerationDraft<string>(),
  };
  const listeners = new Set<() => void>();
  const referencePreviews = new Map<string, string>();
  const ownedReferenceUrls = new Set<string>();
  const retainedReferenceUrls = new Set<string>();
  const deferredRevocations = new Set<string>();

  function emit(nextSnapshot: QuickGenerationSessionSnapshot) {
    snapshot = nextSnapshot;
    for (const listener of listeners) listener();
  }

  function applyAsset(asset?: QuickGenerationAsset) {
    emit({
      currentAssetId: asset?.id ?? null,
      draft: createQuickGenerationDraft(
        asset,
        asset ? (referencePreviews.get(asset.id) ?? "") : "",
      ),
    });
  }

  function forceRevokeOwnedUrl(url: string) {
    if (!ownedReferenceUrls.has(url)) return;
    referencePreview.revokePreviewUrl(url);
    ownedReferenceUrls.delete(url);
    deferredRevocations.delete(url);
  }

  function revokeOwnedUrl(url: string) {
    if (!ownedReferenceUrls.has(url)) return;
    if (retainedReferenceUrls.has(url)) {
      deferredRevocations.add(url);
      return;
    }
    forceRevokeOwnedUrl(url);
  }

  function committedReferenceForCurrentAsset() {
    return snapshot.currentAssetId
      ? referencePreviews.get(snapshot.currentAssetId)
      : undefined;
  }

  function releaseUncommittedReference() {
    const reference = snapshot.draft.reference;
    if (reference && reference !== committedReferenceForCurrentAsset()) {
      revokeOwnedUrl(reference);
    }
  }

  function commitReferencePreview(assetId: string, previewUrl: string) {
    const previousPreview = referencePreviews.get(assetId);
    if (previousPreview && previousPreview !== previewUrl) {
      revokeOwnedUrl(previousPreview);
    }
    if (previewUrl) {
      referencePreviews.set(assetId, previewUrl);
    } else {
      referencePreviews.delete(assetId);
    }
  }

  function releaseReferencePreview(assetId: string) {
    const previewUrl = referencePreviews.get(assetId);
    if (!previewUrl) return;
    revokeOwnedUrl(previewUrl);
    referencePreviews.delete(assetId);
  }

  function releaseRetainedReference(url: string, keep: boolean) {
    if (!url || !ownedReferenceUrls.has(url)) return;
    retainedReferenceUrls.delete(url);
    if (keep) {
      deferredRevocations.delete(url);
      return;
    }
    if (
      deferredRevocations.has(url) ||
      (snapshot.draft.reference !== url &&
        ![...referencePreviews.values()].includes(url))
    ) {
      forceRevokeOwnedUrl(url);
    }
  }

  return {
    subscribe: (listener) => {
      listeners.add(listener);
      return () => listeners.delete(listener);
    },
    getSnapshot: () => snapshot,
    synchronize: (assets) => {
      if (!initialized) {
        initialized = true;
        applyAsset(assets[0]);
        return;
      }
      if (
        snapshot.currentAssetId &&
        !assets.some((asset) => asset.id === snapshot.currentAssetId)
      ) {
        applyAsset(assets[0]);
      }
    },
    selectAsset: (asset) => {
      initialized = true;
      releaseUncommittedReference();
      applyAsset(asset);
    },
    startNewAsset: () => {
      initialized = true;
      releaseUncommittedReference();
      applyAsset();
    },
    updateDraft: (patch) => {
      emit({ ...snapshot, draft: { ...snapshot.draft, ...patch } });
    },
    chooseReference: (file) => {
      if (!file?.type.startsWith("image/")) return;
      releaseUncommittedReference();
      const previewUrl = referencePreview.createPreviewUrl(file);
      ownedReferenceUrls.add(previewUrl);
      emit({
        ...snapshot,
        draft: {
          ...snapshot.draft,
          reference: previewUrl,
          referenceFileName: file.name,
        },
      });
    },
    clearReference: () => {
      releaseUncommittedReference();
      emit({
        ...snapshot,
        draft: {
          ...snapshot.draft,
          reference: "",
          referenceFileName: undefined,
        },
      });
    },
    prepareGeneration: () => {
      const input = toGenerateQuickAssetInput(snapshot.draft);
      if (!input) return undefined;
      const submittedReference = snapshot.draft.reference ?? "";
      if (ownedReferenceUrls.has(submittedReference)) {
        retainedReferenceUrls.add(submittedReference);
      }
      let settled = false;

      return {
        input,
        complete: (asset) => {
          if (settled) return;
          settled = true;
          releaseUncommittedReference();
          commitReferencePreview(asset.id, submittedReference);
          releaseRetainedReference(submittedReference, true);
          applyAsset(asset);
        },
        fail: () => {
          if (settled) return;
          settled = true;
          releaseRetainedReference(submittedReference, false);
        },
      };
    },
    prepareDeletion: (assets) => {
      if (!snapshot.currentAssetId) return undefined;
      const deletedAssetId = snapshot.currentAssetId;
      if (!assets.some((asset) => asset.id === deletedAssetId))
        return undefined;
      const remaining = assets.filter((asset) => asset.id !== deletedAssetId);
      let completed = false;

      return {
        assetId: deletedAssetId,
        complete: () => {
          if (completed) return;
          completed = true;
          if (snapshot.currentAssetId === deletedAssetId) {
            releaseUncommittedReference();
            releaseReferencePreview(deletedAssetId);
            applyAsset(remaining[0]);
          } else {
            releaseReferencePreview(deletedAssetId);
          }
        },
      };
    },
    dispose: () => {
      for (const url of ownedReferenceUrls) {
        referencePreview.revokePreviewUrl(url);
      }
      ownedReferenceUrls.clear();
      retainedReferenceUrls.clear();
      deferredRevocations.clear();
      referencePreviews.clear();
    },
  };
}
