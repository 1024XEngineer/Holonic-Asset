import {
  createQuickGenerationDraft,
  toGenerateQuickAssetInput,
  type GenerateQuickAssetInput,
  type QuickGenerationAsset,
  type QuickGenerationDraft,
} from "@/model";
import {
  browserReferencePreviewAdapter,
  createReferencePreviewLifecycle,
  type ReferencePreviewAdapter,
} from "./reference-preview-lifecycle";

type QuickGenerationDraftPatch = Pick<
  QuickGenerationDraft<string>,
  "prompt" | "size"
>;

export type QuickGenerationSessionSnapshot = {
  currentAssetId: string | null;
  draft: QuickGenerationDraft<string>;
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

export function createQuickGenerationSession(
  referencePreview: ReferencePreviewAdapter = browserReferencePreviewAdapter,
): QuickGenerationSession {
  let initialized = false;
  let snapshot: QuickGenerationSessionSnapshot = {
    currentAssetId: null,
    draft: createQuickGenerationDraft<string>(),
  };
  const listeners = new Set<() => void>();
  const referencePreviews = createReferencePreviewLifecycle(referencePreview);

  function emit(nextSnapshot: QuickGenerationSessionSnapshot) {
    snapshot = nextSnapshot;
    for (const listener of listeners) listener();
  }

  function applyAsset(asset?: QuickGenerationAsset) {
    emit({
      currentAssetId: asset?.id ?? null,
      draft: createQuickGenerationDraft(
        asset,
        asset ? (referencePreviews.previewForAsset(asset.id) ?? "") : "",
      ),
    });
  }

  function releaseUncommittedReference() {
    referencePreviews.releaseUncommitted(
      snapshot.draft.reference,
      snapshot.currentAssetId,
    );
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
      const previewUrl = referencePreviews.create(file);
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
      referencePreviews.retainForSubmission(submittedReference);
      let settled = false;

      return {
        input,
        complete: (asset) => {
          if (settled) return;
          settled = true;
          releaseUncommittedReference();
          referencePreviews.commit(asset.id, submittedReference);
          referencePreviews.settleSubmission(
            submittedReference,
            true,
            snapshot.draft.reference,
          );
          applyAsset(asset);
        },
        fail: () => {
          if (settled) return;
          settled = true;
          referencePreviews.settleSubmission(
            submittedReference,
            false,
            snapshot.draft.reference,
          );
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
            referencePreviews.releaseAsset(deletedAssetId);
            applyAsset(remaining[0]);
          } else {
            referencePreviews.releaseAsset(deletedAssetId);
          }
        },
      };
    },
    dispose: () => {
      referencePreviews.dispose();
    },
  };
}
