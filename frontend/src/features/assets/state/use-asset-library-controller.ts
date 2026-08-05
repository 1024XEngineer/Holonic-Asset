import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import {
  assetKinds,
  useAssetLibraryQuery,
  useCopyAssetMutation,
  useDeleteAssetMutation,
  useUpdateAssetMutation,
} from "@/model/asset";
import { useGenerationRunsQuery } from "@/model/generation";
import type { AssetKind, AssetMetadataUpdate } from "@/model/asset";
import type { GenerationRun } from "@/model/generation";
import type { ProjectSummary } from "@/model/project";

import { getAssetKindConfig } from "../asset-kind-config";
import type { AssetLibraryItem } from "../types/asset";
import { useAssetLibrary } from "./use-asset-library";

export type AssetLibraryController = {
  project?: ProjectSummary;
  query: string;
  selectedKinds: AssetKind[];
  counts: Record<AssetKind, number>;
  totalAssets: number;
  filteredAssets: AssetLibraryItem[];
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
  const copyMutation = useCopyAssetMutation();
  const deleteMutation = useDeleteAssetMutation();
  const updateMutation = useUpdateAssetMutation();
  const { mutateAsync: mutateCopyAsset } = copyMutation;
  const { mutateAsync: mutateDeleteAsset } = deleteMutation;
  const { mutateAsync: mutateUpdateAsset, reset: resetUpdateAsset } =
    updateMutation;
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
  const generationRuns = useMemo(
    () => generationsQuery.data ?? [],
    [generationsQuery.data],
  );
  const {
    counts,
    filteredAssets,
    selectedKinds,
    setSelectedKinds,
    totalAssets,
  } = useAssetLibrary(assetGroups, query);
  const allAssets = useMemo(
    () => assetGroups.flatMap((group) => group.assets),
    [assetGroups],
  );
  const editingAsset = useMemo(() => {
    if (!editingAssetId) return undefined;

    const item = filteredAssets.find((asset) => asset.id === editingAssetId);
    if (item) return item;

    const asset = allAssets.find(
      (candidate) => candidate.id === editingAssetId,
    );
    if (!asset) return undefined;

    const group = assetGroups.find((candidate) =>
      candidate.assets.some((groupAsset) => groupAsset.id === editingAssetId),
    );
    if (!group) return undefined;

    const config = getAssetPresentation(group.kind);
    return { ...asset, kind: group.kind, ...config };
  }, [allAssets, assetGroups, editingAssetId, filteredAssets]);

  useEffect(() => {
    setQuery("");
    setEditingAssetId(undefined);
  }, [projectId]);

  useEffect(() => {
    if (editingAssetId && !editingAsset) setEditingAssetId(undefined);
  }, [editingAsset, editingAssetId]);

  const copyAsset = useCallback(
    (assetId: string) => {
      if (!projectId || !markAssetCopying(assetId)) return;

      void mutateCopyAsset({ projectId, assetId })
        .catch(() => undefined)
        .finally(() => unmarkAssetCopying(assetId));
    },
    [markAssetCopying, mutateCopyAsset, projectId, unmarkAssetCopying],
  );

  const deleteAsset = useCallback(
    (assetId: string) => {
      if (!projectId || !markAssetDeleting(assetId)) return;
      if (editingAssetId === assetId) setEditingAssetId(undefined);

      void mutateDeleteAsset({ projectId, assetId })
        .catch(() => undefined)
        .finally(() => unmarkAssetDeleting(assetId));
    },
    [
      markAssetDeleting,
      mutateDeleteAsset,
      projectId,
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

      void mutateUpdateAsset({ projectId, assetId, metadata })
        .then(() => {
          setEditingAssetId((currentId) =>
            currentId === assetId ? undefined : currentId,
          );
        })
        .catch(() => undefined);
    },
    [editingAssetId, mutateUpdateAsset, projectId],
  );

  return {
    project,
    query,
    selectedKinds,
    counts,
    totalAssets,
    filteredAssets,
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

function getAssetPresentation(kind: AssetKind) {
  const config = getAssetKindConfig(kind);
  return {
    accentClassName: config.accentClassName,
    kindLabel: config.label,
  };
}
