import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  createStore: vi.fn(),
  dispatchCommand: vi.fn(),
  getSnapshot: vi.fn(),
  refValues: [] as unknown[],
  saveRevision: vi.fn(),
  saveSession: vi.fn(),
  saveStateValue: undefined as unknown,
  stateSetters: [] as ReturnType<typeof vi.fn>[],
  store: {
    getState: vi.fn(),
    subscribe: vi.fn(),
    temporal: { getState: vi.fn(), subscribe: vi.fn() },
  },
  useStore: vi.fn(),
}));

vi.mock("react", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react")>();
  return {
    ...actual,
    useCallback: (callback: unknown) => callback,
    useRef: (initial: unknown) => ({
      current: mocks.refValues.length > 0 ? mocks.refValues.shift() : initial,
    }),
    useState: (initial: unknown) => {
      let current =
        mocks.saveStateValue !== undefined
          ? mocks.saveStateValue
          : typeof initial === "function"
            ? (initial as () => unknown)()
            : initial;
      const setter = vi.fn((next: unknown) => {
        current =
          typeof next === "function"
            ? (next as (value: unknown) => unknown)(current)
            : next;
      });
      mocks.stateSetters.push(setter);
      return [current, setter];
    },
  };
});

vi.mock("zustand", () => ({ useStore: mocks.useStore }));

vi.mock("@/model", () => ({
  useSaveAssetRevisionMutation: () => ({ mutateAsync: mocks.saveRevision }),
}));

vi.mock("./editor-session-save", () => ({
  saveEditorSessionRevision: mocks.saveSession,
}));

vi.mock("./editor-session-store", () => ({
  createEditorSessionStore: mocks.createStore,
  dispatchEditorCommand: mocks.dispatchCommand,
  getEditorSessionSnapshot: mocks.getSnapshot,
}));

import { useEditorSession } from "./use-editor-session";

const record = {
  mode: "audio" as const,
  prompt: "Theme",
  audio: {},
};

beforeEach(() => {
  vi.clearAllMocks();
  mocks.refValues.length = 0;
  mocks.stateSetters.length = 0;
  mocks.saveStateValue = undefined;
  mocks.createStore.mockReturnValue(mocks.store);
  mocks.getSnapshot.mockReturnValue({ record, dirty: false });
  mocks.saveRevision.mockResolvedValue(undefined);
  mocks.saveSession.mockImplementation(async (options) => {
    expect(options.isActive()).toBe(true);
    await options.saveRevision(record);
    return { status: "saved" };
  });
});

describe("useEditorSession", () => {
  it("creates a store and dispatches commands", () => {
    mocks.saveStateValue = {
      store: mocks.store,
      state: { phase: "failed", message: "Previous failure" },
    };
    const session = useEditorSession({
      target: { projectId: "project-1", assetId: "asset-1" },
      initialRecord: record,
    });

    session.dispatch({ type: "prompt.set", value: "Updated" });

    expect(mocks.createStore).toHaveBeenCalledWith(record);
    expect(mocks.useStore).toHaveBeenCalledTimes(4);
    expect(mocks.dispatchCommand).toHaveBeenCalledWith(mocks.store, {
      type: "prompt.set",
      value: "Updated",
    });
    expect(mocks.stateSetters[0]).toHaveBeenCalled();
    expect(session.snapshot).toEqual({ record, dirty: false });
  });

  it("reuses a matching session store", () => {
    mocks.refValues.push({
      identity: "project-1\0asset-1",
      store: mocks.store,
    });

    useEditorSession({
      target: { projectId: "project-1", assetId: "asset-1" },
      initialRecord: record,
    });

    expect(mocks.createStore).not.toHaveBeenCalled();
  });

  it("saves the active record and clears saved state", async () => {
    const session = useEditorSession({
      target: { projectId: "project-1", assetId: "asset-1" },
      initialRecord: record,
    });

    await expect(session.save()).resolves.toEqual({ status: "saved" });

    expect(mocks.saveRevision).toHaveBeenCalledWith({
      projectId: "project-1",
      assetId: "asset-1",
      record,
    });
    expect(mocks.stateSetters[0]).toHaveBeenCalledWith({
      store: mocks.store,
      state: { phase: "saving" },
    });
    expect(mocks.stateSetters[0]).toHaveBeenCalledWith({
      store: mocks.store,
      state: { phase: "idle" },
    });
  });

  it("records failed saves and leaves superseded saves unchanged", async () => {
    mocks.saveSession
      .mockResolvedValueOnce({ status: "failed", message: "Save failed" })
      .mockResolvedValueOnce({ status: "superseded" });
    const failedSession = useEditorSession({
      target: { projectId: "project-1", assetId: "asset-1" },
      initialRecord: record,
    });
    await failedSession.save();
    expect(mocks.stateSetters[0]).toHaveBeenCalledWith({
      store: mocks.store,
      state: { phase: "failed", message: "Save failed" },
    });

    mocks.stateSetters.length = 0;
    const supersededSession = useEditorSession({
      target: { projectId: "project-1", assetId: "asset-1" },
      initialRecord: record,
    });
    await supersededSession.save();
    expect(mocks.stateSetters[0]).toHaveBeenCalledTimes(1);
  });

  it("uses idle state when the saved state belongs to another store", () => {
    mocks.saveStateValue = {
      store: { different: true },
      state: { phase: "failed", message: "Stale" },
    };
    useEditorSession({
      target: { projectId: "project-1", assetId: "asset-1" },
      initialRecord: record,
    });

    expect(mocks.getSnapshot).toHaveBeenCalledWith(mocks.store, {
      phase: "idle",
    });
  });
});
