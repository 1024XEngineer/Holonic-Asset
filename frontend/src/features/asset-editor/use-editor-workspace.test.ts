import { beforeEach, describe, expect, it, vi } from "vitest";

import type { AssetRecord, AssetWorkspaceData } from "@/model";

import type { EditorGenerationTask } from "./Header/editor-header";

const mocks = vi.hoisted(() => ({
  animationMutation: {
    isPending: false,
    mutateAsync: vi.fn(),
  },
  applicationMutation: {
    isPending: false,
    mutateAsync: vi.fn(),
  },
  coreCreate: vi.fn(),
  generationRuns: [] as Array<{
    id: string;
    name: string;
    prompt: string;
    status: string;
    error?: string;
  }>,
  schedules: [] as Array<{ callback: () => void; delay: number }>,
  session: {
    dispatch: vi.fn(),
    save: vi.fn(),
    syncExternalRecord: vi.fn(),
    snapshot: {} as {
      record: AssetRecord;
      dirty: boolean;
      canUndo: boolean;
      canRedo: boolean;
      saveState: { phase: "idle" | "saving" | "failed"; message?: string };
    },
  },
  stateValues: [] as unknown[],
}));

vi.mock("react", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react")>();
  return {
    ...actual,
    useEffect: (effect: () => void) => effect(),
    useMemo: (factory: () => unknown) => factory(),
    useState: (initial: unknown) => {
      let current =
        mocks.stateValues.length > 0
          ? mocks.stateValues.shift()
          : typeof initial === "function"
            ? (initial as () => unknown)()
            : initial;
      const setter = vi.fn((next: unknown) => {
        current =
          typeof next === "function"
            ? (next as (value: unknown) => unknown)(current)
            : next;
      });
      return [current, setter];
    },
  };
});

vi.mock("@/hooks/use-timeout", () => ({
  useTimeout: () => ({
    schedule: (callback: () => void, delay: number) => {
      mocks.schedules.push({ callback, delay });
      callback();
    },
  }),
}));

vi.mock("@/model", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/model")>();
  return {
    ...actual,
    coreGenerationApi: { create: mocks.coreCreate },
    useGenerateAnimationMutation: () => mocks.animationMutation,
    useResolveGenerationApplicationMutation: () => mocks.applicationMutation,
    useGenerationRunsQuery: () => ({ data: mocks.generationRuns }),
  };
});

vi.mock("./state", () => ({
  useEditorSession: () => mocks.session,
}));

import { useEditorWorkspace } from "./use-editor-workspace";

beforeEach(() => {
  vi.clearAllMocks();
  mocks.schedules.length = 0;
  mocks.stateValues.length = 0;
  mocks.generationRuns = [];
  mocks.animationMutation.isPending = false;
  mocks.applicationMutation.isPending = false;
  mocks.applicationMutation.mutateAsync.mockResolvedValue(undefined);
  mocks.animationMutation.mutateAsync.mockResolvedValue({
    generationId: "generation-1",
    animation: { kind: "clip", label: "Walk", frameCount: 4 },
  });
  mocks.coreCreate.mockResolvedValue({});
  mocks.session.save.mockResolvedValue({ status: "saved" });
  mocks.session.snapshot = snapshot(spriteRecord("character"));
});

describe("useEditorWorkspace", () => {
  it("returns null for records without a sprite editor", () => {
    mocks.session.snapshot = snapshot({
      mode: "audio",
      prompt: "Theme",
      audio: {},
    });
    mocks.stateValues.push(null, null);

    expect(
      useEditorWorkspace({
        data: workspace(mocks.session.snapshot.record),
        onBack: vi.fn(),
      }),
    ).toBeNull();
  });

  it("maps active tasks and dispatches editor commands", async () => {
    mocks.generationRuns = [
      { id: "pending", name: "Pending", prompt: "One", status: "pending" },
      {
        id: "processing",
        name: "Processing",
        prompt: "Two",
        status: "processing",
      },
      { id: "done", name: "Done", prompt: "Three", status: "completed" },
      {
        id: "failed",
        name: "Failed",
        prompt: "Four",
        status: "failed",
        error: "Video provider rejected the request",
      },
    ];
    mocks.stateValues.push(null, null, null);
    const onBack = vi.fn();
    const editor = useEditorWorkspace({
      data: workspace(mocks.session.snapshot.record),
      onBack,
    });
    expect(editor).not.toBeNull();
    if (!editor) return;

    editor.header.onBack();
    editor.header.onUndo();
    editor.header.onRedo();
    editor.header.onSave();
    editor.sprite.onPositionChange("prototype", { x: 12, y: 18 });
    editor.tree.onAnimationRename("walk", "Walking");
    editor.tree.onAnimationDelete("walk");
    editor.tree.onAnimationGenerate(animationRequest());
    editor.inspector.onPromptChange("New prompt");
    await editor.inspector.onSubmit({
      prompt: "Refine hero",
      target: { nodeIds: ["prototype"], frames: [] },
    });
    await flushPromises();

    expect(editor.header.generationTasks).toEqual([
      {
        id: "pending",
        name: "Pending",
        prompt: "One",
        status: "queued",
      },
      {
        id: "processing",
        name: "Processing",
        prompt: "Two",
        status: "processing",
      },
      {
        id: "failed",
        name: "Failed",
        prompt: "Four",
        status: "failed",
        error: "Video provider rejected the request",
      },
    ]);
    expect(onBack).toHaveBeenCalledOnce();
    expect(mocks.session.save).toHaveBeenCalledOnce();
    expect(mocks.animationMutation.mutateAsync).toHaveBeenCalledWith(
      expect.objectContaining({
        projectId: "7",
        assetId: "8",
        assetKind: "character",
      }),
    );
    expect(mocks.coreCreate).toHaveBeenCalledWith(
      7,
      expect.objectContaining({ assetId: 8, kind: "edit_character_prototype" }),
    );
    expect(mocks.schedules.map(({ delay }) => delay)).toContain(2400);
    expect(mocks.schedules.map(({ delay }) => delay)).toContain(1800);
  });

  it("starts the inspector edit prompt empty instead of using the asset description", () => {
    mocks.stateValues.push(null, null);
    const editor = useEditorWorkspace({
      data: workspace(mocks.session.snapshot.record),
      onBack: vi.fn(),
    });

    expect(editor?.inspector.prompt).toBe("");
    editor?.inspector.onPromptChange("Adjust the silhouette");
    expect(mocks.session.dispatch).not.toHaveBeenCalledWith({
      type: "prompt.set",
      value: "Adjust the silhouette",
    });
  });

  it("includes local tasks and blocks duplicate prompt submission", async () => {
    const animationTask: EditorGenerationTask = {
      id: "animation-local",
      name: "Jump",
      prompt: "Jump",
      status: "processing",
    };
    const promptTask: EditorGenerationTask = {
      id: "prompt-local",
      name: "Edit hero",
      prompt: "Edit",
      status: "processing",
    };
    mocks.stateValues.push(animationTask, promptTask, "Working");

    const editor = useEditorWorkspace({
      data: workspace(mocks.session.snapshot.record),
      onBack: vi.fn(),
    });
    if (!editor) return;
    await editor.inspector.onSubmit({
      prompt: "Ignored",
      target: { nodeIds: [], frames: [] },
    });

    expect(editor.header.generationTasks).toEqual([animationTask, promptTask]);
    expect(editor.inspector.isSubmitting).toBe(true);
    expect(mocks.coreCreate).not.toHaveBeenCalled();
  });

  it("lets the editor apply an awaiting generation result", async () => {
    mocks.generationRuns = [
      {
        id: "ready",
        name: "Walk",
        prompt: "A relaxed walk",
        status: "awaiting_application",
      },
    ];
    mocks.stateValues.push(null, null, null);
    const editor = useEditorWorkspace({
      data: workspace(mocks.session.snapshot.record),
      onBack: vi.fn(),
    });

    editor?.header.generationTasks[0]?.onApply?.();
    await flushPromises();

    expect(mocks.applicationMutation.mutateAsync).toHaveBeenCalledWith({
      projectId: "7",
      assetId: "8",
      runId: "ready",
      applied: true,
    });
  });

  it("reports failed saves, animation generation, and prompt submission", async () => {
    mocks.session.snapshot = {
      ...snapshot(spriteRecord("object")),
      saveState: { phase: "failed", message: "Previous failure" },
    };
    mocks.session.save.mockResolvedValue({
      status: "failed",
      message: "save failed",
    });
    mocks.animationMutation.mutateAsync.mockRejectedValue(
      new Error("animation failed"),
    );
    mocks.coreCreate.mockRejectedValue(new Error("prompt failed"));
    mocks.stateValues.push(null, null, null);
    const editor = useEditorWorkspace({
      data: workspace(mocks.session.snapshot.record),
      onBack: vi.fn(),
    });
    if (!editor) return;

    editor.header.onSave();
    editor.tree.onAnimationGenerate(
      animationRequest({
        animationName: "Open",
        creativeBrief: "Open chest",
      }),
    );
    await expect(
      editor.inspector.onSubmit({
        prompt: "Refine chest",
        target: { nodeIds: [], frames: [] },
      }),
    ).rejects.toThrow("prompt failed");
    await flushPromises();
  });

  it("skips save and remote prompt creation for clean non-numeric assets", async () => {
    mocks.session.snapshot = {
      ...snapshot(spriteRecord("character")),
      dirty: false,
    };
    mocks.stateValues.push(null, null, null);
    const editor = useEditorWorkspace({
      data: workspace(mocks.session.snapshot.record, "project-x", "asset-x"),
      onBack: vi.fn(),
    });
    if (!editor) return;

    editor.header.onSave();
    await editor.inspector.onSubmit({
      prompt: "Local only",
      target: { nodeIds: [], frames: [] },
    });

    expect(mocks.session.save).not.toHaveBeenCalled();
    expect(mocks.coreCreate).not.toHaveBeenCalled();
  });
});

function spriteRecord(mode: "character" | "object"): AssetRecord {
  const data = {
    prototype: {
      format: "png-sprite-sheet" as const,
      imageUrl: "/sprite.png",
      frameWidth: 32,
      frameHeight: 32,
      columns: 4,
      rows: 1,
    },
    animations: [],
    nodePositions: {},
  };
  return mode === "character"
    ? { mode, prompt: "Hero", character: data }
    : { mode, prompt: "Chest", object: data };
}

function snapshot(record: AssetRecord) {
  return {
    record,
    dirty: true,
    canUndo: true,
    canRedo: true,
    saveState: { phase: "idle" as const },
  };
}

function workspace(
  record: AssetRecord,
  projectId = "7",
  assetId = "8",
): AssetWorkspaceData {
  return {
    projectName: "Demo",
    asset: {
      id: assetId,
      projectId,
      kind: record.mode,
      name: "Asset",
      perspective: "Top-Down",
      version: "v1",
      history: [],
    },
    record,
  };
}

async function flushPromises() {
  await Promise.resolve();
  await Promise.resolve();
}

function animationRequest(overrides: Record<string, unknown> = {}) {
  return {
    animationName: "Walk",
    direction: "front" as const,
    creativeBrief: "Walk north",
    frameCount: 8,
    fps: 12,
    duration: 5,
    ...overrides,
  };
}
