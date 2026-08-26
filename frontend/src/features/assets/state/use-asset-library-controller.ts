import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import {
  assetKinds,
  createAssetLibraryCollection,
  mergeAssetTags,
  useAssetLibraryQuery,
  useCopyAssetMutation,
  useDeleteAssetMutation,
  useUpdateAssetMutation,
} from "@/model/asset";
import { useGenerationRunsQuery } from "@/model/generation";
import {
  useCreateProjectTagMutation,
  useProjectTagsQuery,
  useUpdateProjectTagMutation,
} from "@/model/project";
import type {
  AssetKind,
  AssetLibraryItem,
  AssetMetadataUpdate,
  AssetTag,
} from "@/model/asset";
import type { GenerationRun } from "@/model/generation";
import type { ProjectSummary } from "@/model/project";

import { useAssetLibrary } from "./use-asset-library";

export type AssetLibraryController = {
  project?: ProjectSummary;
  query: string;
  selectedKinds: AssetKind[];
  counts: Record<AssetKind, number>;
  totalAssets: number;
  filteredAssets: AssetLibraryItem[];
  availableTags: AssetTag[];
  createTag: (tag: AssetTag) => Promise<AssetTag | undefined>;
  updateTag: (
    currentTag: AssetTag,
    nextTag: AssetTag,
  ) => Promise<AssetTag | undefined>;
  generationRuns: GenerationRun[];
  editingAsset?: AssetLibraryItem;
  actionError?: Error;
  error?: Error;
  isLoading: boolean;
  isUpdatingAsset: boolean;
  updateError?: Error;
  copyingAssetIds: ReadonlySet<string>;
  deletingAssetIds: ReadonlySet<string>;
  clearFilters: () => void;
  closeAssetEditor: () => void;
  copyAsset: (assetId: string) => void;
  deleteAsset: (assetId: string) => void;
  openAssetEditor: (assetId: string) => void;
  retry: () => void;
  updateAsset: (metadata: AssetMetadataUpdate) => void;
  setQuery: (query: string) => void;
  setSelectedKinds: (kinds: AssetKind[]) => void;
};

export function useAssetLibraryController({
  project,
}: {
  project?: ProjectSummary;
}): AssetLibraryController {
  const [query, setQuery] = useState("");
  const [editingAssetId, setEditingAssetId] = useState<string>();
  const projectId = project?.id;
  const assetsQuery = useAssetLibraryQuery(projectId);
  const generationsQuery = useGenerationRunsQuery(projectId);
  const projectTagsQuery = useProjectTagsQuery(projectId);
  const copyMutation = useCopyAssetMutation();
  const deleteMutation = useDeleteAssetMutation();
  const updateMutation = useUpdateAssetMutation();
  const createTagMutation = useCreateProjectTagMutation();
  const updateTagMutation = useUpdateProjectTagMutation();
  const {
    isError: hasCopyError,
    mutateAsync: mutateCopyAsset,
    reset: resetCopyAsset,
  } = copyMutation;
  const {
    isError: hasDeleteError,
    mutateAsync: mutateDeleteAsset,
    reset: resetDeleteAsset,
  } = deleteMutation;
  const { mutateAsync: mutateUpdateAsset, reset: resetUpdateAsset } =
    updateMutation;
  const resetActionErrors = useCallback(() => {
    if (hasCopyError) resetCopyAsset();
    if (hasDeleteError) resetDeleteAsset();
  }, [hasCopyError, hasDeleteError, resetCopyAsset, resetDeleteAsset]);
  const {
    assetIds: copyingAssetIds,
    add: markAssetCopying,
    remove: unmarkAssetCopying,
  } = usePendingAssetIds();
  const {
    assetIds: deletingAssetIds,
    add: markAssetDeleting,
    remove: unmarkAssetDeleting,
  } = usePendingAssetIds();
  const { refetch: refetchAssets } = assetsQuery;
  const assetGroups = useMemo(() => assetsQuery.data ?? [], [assetsQuery.data]);
  const assetCollection = useMemo(
    () => createAssetLibraryCollection(assetGroups),
    [assetGroups],
  );
  const generationRuns = useMemo(
    () => generationsQuery.data ?? [],
    [generationsQuery.data],
  );
  const availableTags = useMemo(
    () =>
      mergeAssetTags(
        projectTagsQuery.data ?? [],
        ...assetCollection.items.map((asset) => asset.tags),
      ),
    [assetCollection, projectTagsQuery.data],
  );
  const {
    counts,
    filteredAssets,
    selectedKinds,
    setSelectedKinds,
    totalAssets,
  } = useAssetLibrary(assetCollection, query);
  const editingAsset = useMemo(() => {
    if (!editingAssetId) return undefined;

    return (
      filteredAssets.find((asset) => asset.id === editingAssetId) ??
      assetCollection.find(editingAssetId)
    );
  }, [assetCollection, editingAssetId, filteredAssets]);

  useEffect(() => {
    resetCopyAsset();
    resetDeleteAsset();
    setQuery("");
    setSelectedKinds([...assetKinds]);
    setEditingAssetId(undefined);
  }, [projectId, resetCopyAsset, resetDeleteAsset, setSelectedKinds]);

  useEffect(() => {
    if (editingAssetId && !editingAsset) setEditingAssetId(undefined);
  }, [editingAsset, editingAssetId]);

  const copyAsset = useCallback(
    (assetId: string) => {
      if (!projectId || !markAssetCopying(assetId)) return;
      resetActionErrors();

      void (async () => {
        try {
          await mutateCopyAsset({ projectId, assetId });
        } catch {
          // The mutation exposes its error through controller state.
        } finally {
          unmarkAssetCopying(assetId);
        }
      })();
    },
    [
      markAssetCopying,
      mutateCopyAsset,
      projectId,
      resetActionErrors,
      unmarkAssetCopying,
    ],
  );

  const deleteAsset = useCallback(
    (assetId: string) => {
      if (!projectId || !markAssetDeleting(assetId)) return;
      resetActionErrors();
      if (editingAssetId === assetId) setEditingAssetId(undefined);

      void (async () => {
        try {
          await mutateDeleteAsset({ projectId, assetId });
        } catch {
          // The mutation exposes its error through controller state.
        } finally {
          unmarkAssetDeleting(assetId);
        }
      })();
    },
    [
      markAssetDeleting,
      mutateDeleteAsset,
      projectId,
      resetActionErrors,
      editingAssetId,
      unmarkAssetDeleting,
    ],
  );

  const clearFilters = useCallback(() => {
    setQuery("");
    setSelectedKinds([...assetKinds]);
  }, [setSelectedKinds]);

  const retry = useCallback(() => {
    void refetchAssets();
  }, [refetchAssets]);

  const closeAssetEditor = useCallback(() => {
    resetUpdateAsset();
    setEditingAssetId(undefined);
  }, [resetUpdateAsset]);

  const openAssetEditor = useCallback(
    (assetId: string) => {
      resetUpdateAsset();
      setEditingAssetId(assetId);
    },
    [resetUpdateAsset],
  );

  const updateAsset = useCallback(
    (metadata: AssetMetadataUpdate) => {
      if (!projectId || !editingAssetId) return;
      const assetId = editingAssetId;

      void (async () => {
        try {
          await mutateUpdateAsset({ projectId, assetId, metadata });
          setEditingAssetId((currentId) =>
            currentId === assetId ? undefined : currentId,
          );
        } catch {
          // The mutation exposes its error through controller state.
        }
      })();
    },
    [editingAssetId, mutateUpdateAsset, projectId],
  );

  const createTag = useCallback(
    (tag: AssetTag) => {
      if (!projectId) return Promise.resolve(undefined);
      return createTagMutation.mutateAsync({ projectId, tag });
    },
    [createTagMutation, projectId],
  );

  const updateTag = useCallback(
    (currentTag: AssetTag, nextTag: AssetTag) => {
      if (!projectId || !currentTag.tagId) return Promise.resolve(nextTag);
      return updateTagMutation.mutateAsync({
        projectId,
        tagId: currentTag.tagId,
        tag: nextTag,
      });
    },
    [projectId, updateTagMutation],
  );

  return {
    project,
    query,
    selectedKinds,
    counts,
    totalAssets,
    filteredAssets,
    availableTags,
    createTag,
    updateTag,
    generationRuns,
    editingAsset,
    actionError: copyMutation.error ?? deleteMutation.error ?? undefined,
    error: assetsQuery.error ?? undefined,
    isLoading: assetsQuery.isPending,
    isUpdatingAsset: updateMutation.isPending,
    updateError: updateMutation.error ?? undefined,
    copyingAssetIds,
    deletingAssetIds,
    clearFilters,
    closeAssetEditor,
    copyAsset,
    deleteAsset,
    openAssetEditor,
    retry,
    updateAsset,
    setQuery,
    setSelectedKinds,
  };
}

function usePendingAssetIds() {
  const [assetIds, setAssetIds] = useState<Set<string>>(() => new Set());
  const assetIdsRef = useRef(assetIds);

  const add = useCallback((assetId: string) => {
    if (assetIdsRef.current.has(assetId)) return false;

    const nextAssetIds = new Set(assetIdsRef.current);
    nextAssetIds.add(assetId);
    assetIdsRef.current = nextAssetIds;
    setAssetIds(nextAssetIds);
    return true;
  }, []);

  const remove = useCallback((assetId: string) => {
    if (!assetIdsRef.current.has(assetId)) return;

    const nextAssetIds = new Set(assetIdsRef.current);
    nextAssetIds.delete(assetId);
    assetIdsRef.current = nextAssetIds;
    setAssetIds(nextAssetIds);
  }, []);

  return { assetIds, add, remove };
}
