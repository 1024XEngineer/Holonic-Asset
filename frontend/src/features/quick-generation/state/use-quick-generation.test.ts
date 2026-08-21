import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  cleanups: [] as Array<() => void>,
  deleteMutation: {
    error: null as Error | null,
    isPending: false,
    mutate: vi.fn(),
  },
  generateMutation: {
    error: null as Error | null,
    isPending: false,
    mutate: vi.fn(),
  },
  query: {
    data: undefined as Array<{ id: string; name: string }> | undefined,
    error: null as Error | null,
    isPending: false,
    isSuccess: false,
    refetch: vi.fn(),
  },
  session: {
    chooseCreatingReference: vi.fn(),
    clearCreatingReference: vi.fn(),
    dispose: vi.fn(),
    getSnapshot: vi.fn(),
    prepareDeletion: vi.fn(),
    prepareGeneration: vi.fn(),
    selectAsset: vi.fn(),
    startNewAsset: vi.fn(),
    subscribe: vi.fn(),
    synchronize: vi.fn(),
    updateDraft: vi.fn(),
  },
}));

vi.mock("react", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react")>();
  return {
    ...actual,
    useEffect: (effect: () => void | (() => void)) => {
      const cleanup = effect();
      if (cleanup) mocks.cleanups.push(cleanup);
    },
    useRef: (value: unknown) => ({ current: value }),
    useSyncExternalStore: (_subscribe: unknown, getSnapshot: () => unknown) =>
      getSnapshot(),
  };
});

vi.mock("@/model", () => ({
  useDeleteQuickAssetMutation: () => mocks.deleteMutation,
  useGenerateQuickAssetMutation: () => mocks.generateMutation,
  useQuickAssetsQuery: () => mocks.query,
}));

vi.mock("./quick-generation-session", () => ({
  createQuickGenerationSession: () => mocks.session,
}));

import { useQuickGeneration } from "./use-quick-generation";

beforeEach(() => {
  vi.clearAllMocks();
  mocks.cleanups.length = 0;
  mocks.query.data = [
    { id: "asset-1", name: "First" },
    { id: "asset-2", name: "Second" },
  ];
  mocks.query.error = null;
  mocks.query.isPending = false;
  mocks.query.isSuccess = true;
  mocks.generateMutation.error = null;
  mocks.generateMutation.isPending = false;
  mocks.deleteMutation.error = null;
  mocks.deleteMutation.isPending = false;
  mocks.session.getSnapshot.mockReturnValue({
    currentAssetId: "asset-1",
    draft: { prompt: "Tree", size: "64 x 64 px" },
  });
});

describe("useQuickGeneration", () => {
  it("synchronizes assets and completes generation and deletion", () => {
    const completeGeneration = vi.fn();
    const failGeneration = vi.fn();
    const completeDeletion = vi.fn();
    mocks.session.prepareGeneration.mockReturnValue({
      input: { prompt: "Tree", size: "64 x 64 px" },
      complete: completeGeneration,
      fail: failGeneration,
    });
    mocks.session.prepareDeletion.mockReturnValue({
      assetId: "asset-1",
      complete: completeDeletion,
    });
    mocks.generateMutation.mutate.mockImplementation((_input, options) =>
      options.onSuccess(),
    );
    mocks.deleteMutation.mutate.mockImplementation((_input, options) =>
      options.onSuccess(),
    );

    const controller = useQuickGeneration();
    controller.actions.generate();
    controller.actions.deleteCurrentAsset();
    mocks.cleanups.forEach((cleanup) => cleanup());

    expect(controller.model.currentAsset).toEqual({
      id: "asset-1",
      name: "First",
    });
    expect(mocks.session.synchronize).toHaveBeenCalledWith(mocks.query.data);
    expect(completeGeneration).toHaveBeenCalledOnce();
    expect(failGeneration).not.toHaveBeenCalled();
    expect(completeDeletion).toHaveBeenCalledOnce();
    expect(mocks.session.dispose).toHaveBeenCalledOnce();
  });

  it("does nothing when session actions cannot prepare a mutation", () => {
    mocks.query.data = undefined;
    mocks.query.isSuccess = false;
    mocks.session.getSnapshot.mockReturnValue({
      currentAssetId: null,
      draft: { prompt: "", size: "64 x 64 px" },
    });
    mocks.session.prepareGeneration.mockReturnValue(undefined);
    mocks.session.prepareDeletion.mockReturnValue(undefined);
    mocks.generateMutation.error = new Error("generation failed");
    mocks.deleteMutation.isPending = true;

    const controller = useQuickGeneration();
    controller.actions.generate();
    controller.actions.deleteCurrentAsset();

    expect(controller.model.currentAsset).toBeUndefined();
    expect(controller.model.status.isMutating).toBe(true);
    expect(controller.model.status.actionError).toBe(
      mocks.generateMutation.error,
    );
    expect(mocks.generateMutation.mutate).not.toHaveBeenCalled();
    expect(mocks.deleteMutation.mutate).not.toHaveBeenCalled();
    expect(mocks.session.synchronize).not.toHaveBeenCalled();
  });
});
