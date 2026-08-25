import { beforeEach, describe, expect, it, vi } from "vitest";

import type { AssetKind, AssetLibraryItem } from "@/model/asset";
import type { ProjectSummary } from "@/model/project";

const mocks = vi.hoisted(() => ({
  assetQuery: {
    data: [] as unknown[],
    error: null as Error | null,
    isPending: false,
    refetch: vi.fn(),
  },
  collection: { find: vi.fn(), items: [] as AssetLibraryItem[] },
  copy: {
    error: null as Error | null,
    isError: false,
    mutateAsync: vi.fn(),
    reset: vi.fn(),
  },
  delete: {
    error: null as Error | null,
    isError: false,
    mutateAsync: vi.fn(),
    reset: vi.fn(),
  },
  generationQuery: { data: undefined as unknown[] | undefined },
  library: {
    counts: {} as Record<AssetKind, number>,
    filteredAssets: [] as AssetLibraryItem[],
    selectedKinds: [] as AssetKind[],
    setSelectedKinds: vi.fn(),
    totalAssets: 0,
  },
  stateSetters: [] as ReturnType<typeof vi.fn>[],
  stateValues: [] as unknown[],
  update: {
    error: null as Error | null,
    isPending: false,
    mutateAsync: vi.fn(),
    reset: vi.fn(),
  },
}));

vi.mock("react", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react")>();
  return {
    ...actual,
    useCallback: (callback: unknown) => callback,
    useEffect: (effect: () => void) => effect(),
    useMemo: (factory: () => unknown) => factory(),
    useRef: (value: unknown) => ({ current: value }),
    useState: (initial: unknown) => {
      const fallback =
        typeof initial === "function" ? (initial as () => unknown)() : initial;
      const value =
        mocks.stateValues.length > 0 ? mocks.stateValues.shift() : fallback;
      const setter = vi.fn();
      mocks.stateSetters.push(setter);
      return [value, setter];
    },
  };
});

vi.mock("@/model/asset", () => ({
  assetKinds: ["character", "object", "tileset", "scenery", "uiset", "audio"],
  createAssetLibraryCollection: () => mocks.collection,
  mergeAssetTags: (...collections: unknown[][]) => collections.flat(),
  useAssetLibraryQuery: () => mocks.assetQuery,
  useCopyAssetMutation: () => mocks.copy,
  useDeleteAssetMutation: () => mocks.delete,
  useUpdateAssetMutation: () => mocks.update,
}));

vi.mock("@/model/generation", () => ({
  useGenerationRunsQuery: () => mocks.generationQuery,
}));

vi.mock("./use-asset-library", () => ({
  useAssetLibrary: () => mocks.library,
}));

import { useAssetLibraryController } from "./use-asset-library-controller";

const project: ProjectSummary = {
  id: "project-1",
  name: "Project",
  style: "Top-Down",
  gameType: "Role-playing game",
  platform: "PC",
  description: "",
  reference: "",
  perspective: "Top-Down",
  assetCount: 1,
};

const asset = {
  id: "asset-1",
  kind: "character",
  name: "Hero",
} as AssetLibraryItem;

beforeEach(() => {
  vi.clearAllMocks();
  mocks.stateSetters.length = 0;
  mocks.stateValues.length = 0;
  mocks.assetQuery.data = [];
  mocks.assetQuery.error = null;
  mocks.assetQuery.isPending = false;
  mocks.generationQuery.data = undefined;
  mocks.collection.find.mockReturnValue(undefined);
  mocks.library.filteredAssets = [];
  mocks.library.selectedKinds = ["character"];
  mocks.library.totalAssets = 0;
  mocks.copy.error = null;
  mocks.copy.isError = false;
  mocks.copy.mutateAsync.mockResolvedValue(undefined);
  mocks.delete.error = null;
  mocks.delete.isError = false;
  mocks.delete.mutateAsync.mockResolvedValue(undefined);
  mocks.update.error = null;
  mocks.update.isPending = false;
  mocks.update.mutateAsync.mockResolvedValue(undefined);
});

describe("useAssetLibraryController", () => {
  it("runs asset actions once while mutations are pending", async () => {
    mocks.stateValues.push("sword", "asset-1", new Set(), new Set());
    mocks.library.filteredAssets = [asset];
    mocks.generationQuery.data = [{ id: "run-1" }];
    mocks.copy.isError = true;
    mocks.delete.isError = true;

    const controller = useAssetLibraryController({ project });
    controller.copyAsset("asset-1");
    controller.copyAsset("asset-1");
    controller.deleteAsset("asset-1");
    controller.deleteAsset("asset-1");
    controller.clearFilters();
    controller.retry();
    controller.openAssetEditor("asset-1");
    controller.closeAssetEditor();
    controller.updateAsset({
      name: "Updated",
      description: "",
      tags: [],
      canvasSize: "64 x 64 px",
      perspective: "Top-Down",
    });
    await flushPromises();

    expect(controller.editingAsset).toBe(asset);
    expect(controller.generationRuns).toEqual([{ id: "run-1" }]);
    expect(mocks.copy.mutateAsync).toHaveBeenCalledOnce();
    expect(mocks.delete.mutateAsync).toHaveBeenCalledOnce();
    expect(mocks.update.mutateAsync).toHaveBeenCalledOnce();
    expect(mocks.copy.reset).toHaveBeenCalled();
    expect(mocks.delete.reset).toHaveBeenCalled();
    expect(mocks.update.reset).toHaveBeenCalledTimes(2);
    expect(mocks.assetQuery.refetch).toHaveBeenCalledOnce();
    expect(mocks.library.setSelectedKinds).toHaveBeenCalled();
  });

  it("falls back to the full collection and exposes query errors", () => {
    mocks.stateValues.push("", "asset-2", new Set(), new Set());
    mocks.collection.find.mockReturnValue(asset);
    mocks.assetQuery.error = new Error("load failed");
    mocks.assetQuery.isPending = true;
    mocks.copy.error = new Error("copy failed");
    mocks.update.error = new Error("update failed");
    mocks.update.isPending = true;

    const controller = useAssetLibraryController({ project });

    expect(controller.editingAsset).toBe(asset);
    expect(controller.error).toBe(mocks.assetQuery.error);
    expect(controller.actionError).toBe(mocks.copy.error);
    expect(controller.updateError).toBe(mocks.update.error);
    expect(controller.isLoading).toBe(true);
    expect(controller.isUpdatingAsset).toBe(true);
  });

  it("ignores mutations without a project or active editor", () => {
    mocks.stateValues.push("", undefined, new Set(), new Set());
    const controller = useAssetLibraryController({ project: undefined });

    controller.copyAsset("asset-1");
    controller.deleteAsset("asset-1");
    controller.updateAsset({
      name: "Updated",
      description: "",
      tags: [],
      canvasSize: "64 x 64 px",
      perspective: "Top-Down",
    });

    expect(controller.editingAsset).toBeUndefined();
    expect(mocks.copy.mutateAsync).not.toHaveBeenCalled();
    expect(mocks.delete.mutateAsync).not.toHaveBeenCalled();
    expect(mocks.update.mutateAsync).not.toHaveBeenCalled();
  });

  it("absorbs rejected mutation promises", async () => {
    mocks.stateValues.push("", "asset-1", new Set(), new Set());
    mocks.library.filteredAssets = [asset];
    mocks.copy.mutateAsync.mockRejectedValue(new Error("copy failed"));
    mocks.delete.mutateAsync.mockRejectedValue(new Error("delete failed"));
    mocks.update.mutateAsync.mockRejectedValue(new Error("update failed"));
    const controller = useAssetLibraryController({ project });

    controller.copyAsset("asset-1");
    controller.deleteAsset("asset-1");
    controller.updateAsset({
      name: "Updated",
      description: "",
      tags: [],
      canvasSize: "64 x 64 px",
      perspective: "Top-Down",
    });
    await flushPromises();

    expect(mocks.copy.mutateAsync).toHaveBeenCalledOnce();
    expect(mocks.delete.mutateAsync).toHaveBeenCalledOnce();
    expect(mocks.update.mutateAsync).toHaveBeenCalledOnce();
  });
});

async function flushPromises() {
  await Promise.resolve();
  await Promise.resolve();
}
