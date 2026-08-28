import { beforeEach, describe, expect, it, vi } from "vitest";

import type {
  AssetRecord,
  AssetWorkspaceData,
  CharacterAnimation,
  GenerationTaskType,
} from "@/model";
import type { GenerationTaskListItem } from "@/features/generation";

const mocks = vi.hoisted(() => ({
  animationMutation: {
    isPending: false,
    mutateAsync: vi.fn(),
  },
  derivationMutation: {
    isPending: false,
    mutateAsync: vi.fn(),
  },
  assetExport: {
    exportAsset: vi.fn(),
    isExporting: false,
  },
  applicationMutation: {
    isPending: false,
    mutateAsync: vi.fn(),
  },
  enqueueAssetEdit: vi.fn(),
  candidateQuery: {
    data: undefined as
      | {
          kind: GenerationTaskType;
          result?: { animation_id?: number; content?: unknown };
          status: string;
        }
      | undefined,
    isError: false,
    isPending: false,
  },
  generationRuns: [] as Array<{
    id: string;
    name: string;
    prompt: string;
    status: string;
    projectId?: string;
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
      let current: unknown;
      if (mocks.stateValues.length > 0) current = mocks.stateValues.shift();
      else if (typeof initial === "function")
        current = (initial as () => unknown)();
      else current = initial;
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
    useEnqueueAssetEditGenerationMutation: () => ({
      mutateAsync: mocks.enqueueAssetEdit,
    }),
    useGenerateAnimationMutation: () => mocks.animationMutation,
    useDeriveAnimationMutation: () => mocks.derivationMutation,
    useAssetExport: () => mocks.assetExport,
    useResolveGenerationApplicationMutation: () => mocks.applicationMutation,
    useGenerationCandidateQuery: () => mocks.candidateQuery,
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
  mocks.derivationMutation.isPending = false;
  mocks.applicationMutation.isPending = false;
  mocks.assetExport.isExporting = false;
  mocks.assetExport.exportAsset.mockResolvedValue(undefined);
  mocks.candidateQuery.data = undefined;
  mocks.candidateQuery.isError = false;
  mocks.candidateQuery.isPending = false;
  mocks.applicationMutation.mutateAsync.mockResolvedValue(undefined);
  mocks.animationMutation.mutateAsync.mockResolvedValue({
    generationId: "generation-1",
    animation: { kind: "clip", label: "Walk", frameCount: 4 },
  });
  mocks.derivationMutation.mutateAsync.mockResolvedValue({
    id: "generation-2",
    name: "Walk",
    prompt: "Derive Walk for front",
  });
  mocks.enqueueAssetEdit.mockResolvedValue({ id: "31" });
  mocks.session.save.mockResolvedValue({ status: "saved" });
  mocks.session.snapshot = snapshot(spriteRecord("character"));
});

describe("useEditorWorkspace", () => {
  it("returns null for records without a sprite editor", () => {
    mocks.session.snapshot = snapshot({
      mode: "uiset",
      prompt: "Inventory",
      uiset: { components: [] },
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
        projectId: "7",
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
        status: "pending",
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
        projectId: "7",
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
    expect(mocks.enqueueAssetEdit).not.toHaveBeenCalled();
    expect(mocks.schedules.map(({ delay }) => delay)).toContain(2400);
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
    const animationTask: GenerationTaskListItem = {
      id: "animation-local",
      name: "Jump",
      prompt: "Jump",
      status: "processing",
    };
    const promptTask: GenerationTaskListItem = {
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
    expect(mocks.enqueueAssetEdit).not.toHaveBeenCalled();
  });

  it("previews an awaiting result in the editor and offers apply or deny", async () => {
    mocks.generationRuns = [
      {
        id: "ready",
        name: "Walk",
        prompt: "A relaxed walk",
        status: "awaiting_application",
      },
    ];
    mocks.candidateQuery.data = {
      kind: "edit_character_prototype",
      status: "awaiting_application",
      result: {
        content: {
          prototype: [
            { id: 1, url: "/candidate-front.png" },
            { id: 2, url: "/candidate-right.png" },
            { id: 3, url: "/candidate-back.png" },
            { id: 4, url: "/candidate-left.png" },
          ],
        },
      },
    };
    mocks.stateValues.push(null, null, null);
    const editor = useEditorWorkspace({
      data: workspace(mocks.session.snapshot.record),
      onBack: vi.fn(),
    });

    expect(editor?.header.generationTasks).toEqual([]);
    expect(editor?.sprite.prototype.imageUrl).toBe("/sprite.png");
    expect(editor?.generationReview).toMatchObject({
      kind: "comparison",
      nodeId: "prototype",
      candidatePrototype: { imageUrl: "/candidate-front.png" },
    });
    editor?.generationReview?.onApply();
    editor?.generationReview?.onDeny();
    await flushPromises();

    expect(mocks.applicationMutation.mutateAsync).toHaveBeenNthCalledWith(1, {
      projectId: "7",
      assetId: "8",
      runId: "ready",
      applied: true,
    });
    expect(mocks.applicationMutation.mutateAsync).toHaveBeenNthCalledWith(2, {
      projectId: "7",
      assetId: "8",
      runId: "ready",
      applied: false,
    });
    expect(mocks.session.dispatch).toHaveBeenCalledTimes(1);
    expect(mocks.session.dispatch).toHaveBeenCalledWith({
      type: "record.candidate.apply",
      record: expect.objectContaining({
        mode: "character",
        character: expect.objectContaining({
          prototype: expect.objectContaining({
            imageUrl: "/candidate-front.png",
          }),
        }),
      }),
    });
    expect(mocks.session.save).not.toHaveBeenCalled();
  });

  it("previews and applies an awaiting frame edit without saving", async () => {
    mocks.session.snapshot = snapshot(spriteRecordWithAnimations("object"));
    mocks.generationRuns = [
      {
        id: "ready",
        name: "Open",
        prompt: "Open more dramatically",
        status: "awaiting_application",
      },
    ];
    mocks.candidateQuery.data = {
      kind: "edit_frames",
      status: "awaiting_application",
      result: {
        animation_id: 42,
        content: {
          animations: [
            {
              id: 42,
              name: "Open",
              frames: [
                { id: 1, url: "/open-edited-1.png" },
                { id: 2, url: "/open-edited-2.png" },
              ],
            },
          ],
        },
      },
    };
    mocks.stateValues.push(null, null, null);

    const editor = useEditorWorkspace({
      data: workspace(mocks.session.snapshot.record),
      onBack: vi.fn(),
    });

    expect(editor?.generationReview).toMatchObject({
      kind: "comparison",
      nodeId: "42",
      candidateAnimation: {
        id: "42",
        label: "Open",
        spriteSheet: {
          frameUrls: ["/open-edited-1.png", "/open-edited-2.png"],
        },
      },
    });

    editor?.generationReview?.onApply();
    await flushPromises();

    expect(mocks.applicationMutation.mutateAsync).toHaveBeenCalledWith({
      projectId: "7",
      assetId: "8",
      runId: "ready",
      applied: true,
    });
    expect(mocks.session.dispatch).toHaveBeenCalledWith({
      type: "record.candidate.apply",
      record: expect.objectContaining({
        mode: "object",
        object: expect.objectContaining({
          animations: [
            expect.objectContaining({
              id: "42",
              spriteSheet: expect.objectContaining({
                frameUrls: ["/open-edited-1.png", "/open-edited-2.png"],
              }),
            }),
            expect.objectContaining({ id: "77", label: "Idle" }),
          ],
        }),
      }),
    });
    expect(mocks.session.save).not.toHaveBeenCalled();
  });

  it("shows a new animation as a reviewable canvas node", () => {
    mocks.generationRuns = [
      {
        id: "ready",
        name: "Walk",
        prompt: "A relaxed walk",
        status: "awaiting_application",
      },
    ];
    mocks.candidateQuery.data = {
      kind: "generate_animation",
      status: "awaiting_application",
      result: {
        content: {
          animations: [
            {
              name: "Walk",
              frames: [{ id: 1, url: "/walk-1.png" }],
            },
          ],
        },
      },
    };
    mocks.stateValues.push(null, null, null);

    const editor = useEditorWorkspace({
      data: workspace(mocks.session.snapshot.record),
      onBack: vi.fn(),
    });

    expect(editor?.sprite.animations).toEqual([
      expect.objectContaining({ id: "1", label: "Walk" }),
    ]);
    expect(editor?.generationReview).toMatchObject({
      kind: "new-animation",
      nodeId: "1",
    });
  });

  it("handles a generation application failure", async () => {
    mocks.generationRuns = [
      {
        id: "ready",
        name: "Walk",
        prompt: "A relaxed walk",
        status: "awaiting_application",
      },
    ];
    mocks.applicationMutation.mutateAsync.mockRejectedValue(
      new Error("application failed"),
    );
    mocks.candidateQuery.data = {
      kind: "edit_character_prototype",
      status: "awaiting_application",
      result: {
        content: {
          prototype: [{ id: 1, url: "/candidate-front.png" }],
        },
      },
    };
    mocks.stateValues.push(null, null, null);
    const editor = useEditorWorkspace({
      data: workspace(mocks.session.snapshot.record),
      onBack: vi.fn(),
    });

    editor?.generationReview?.onApply();
    await flushPromises();

    expect(mocks.applicationMutation.mutateAsync).toHaveBeenCalledWith({
      projectId: "7",
      assetId: "8",
      runId: "ready",
      applied: true,
    });
    expect(mocks.session.dispatch).not.toHaveBeenCalled();
    expect(mocks.schedules.map(({ delay }) => delay)).toContain(2400);
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
    mocks.enqueueAssetEdit.mockRejectedValue(new Error("prompt failed"));
    mocks.stateValues.push(null, null, null);
    const editor = useEditorWorkspace({
      data: workspace(mocks.session.snapshot.record),
      onBack: vi.fn(),
    });
    if (!editor) return;

    editor.header.onSave();
    editor.header.onExport();
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

  it("saves before exporting a persisted asset", async () => {
    const editor = useEditorWorkspace({
      data: workspace(mocks.session.snapshot.record),
      onBack: vi.fn(),
    });
    if (!editor) return;

    editor.header.onExport();
    await flushPromises();

    expect(mocks.session.save).toHaveBeenCalledOnce();
    expect(mocks.assetExport.exportAsset).toHaveBeenCalledWith(8);
  });

  it("reports export failures and skips non-persisted assets", async () => {
    mocks.assetExport.exportAsset.mockRejectedValue(new Error("export failed"));
    const editor = useEditorWorkspace({
      data: workspace(mocks.session.snapshot.record),
      onBack: vi.fn(),
    });
    if (!editor) return;

    editor.header.onExport();
    await flushPromises();
    expect(mocks.assetExport.exportAsset).toHaveBeenCalledWith(8);

    mocks.session.snapshot = {
      ...snapshot(spriteRecord("character")),
      dirty: false,
    };
    const draftEditor = useEditorWorkspace({
      data: workspace(mocks.session.snapshot.record, "project-x", "asset-x"),
      onBack: vi.fn(),
    });
    draftEditor?.header.onExport();
    await flushPromises();

    expect(mocks.assetExport.exportAsset).toHaveBeenCalledOnce();
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
    expect(mocks.enqueueAssetEdit).not.toHaveBeenCalled();
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

function spriteRecordWithAnimations(mode: "character" | "object"): AssetRecord {
  const record = spriteRecord(mode);
  const animations: CharacterAnimation[] = [
    {
      kind: "clip",
      id: "42",
      label: "Open",
      frameCount: 2,
      spriteSheet: {
        format: "png-sprite-sheet",
        imageUrl: "/open-1.png",
        frameUrls: ["/open-1.png", "/open-2.png"],
        frameWidth: 32,
        frameHeight: 32,
        columns: 2,
        rows: 1,
      },
    },
    {
      kind: "clip",
      id: "77",
      label: "Idle",
      frameCount: 1,
      spriteSheet: {
        format: "png-sprite-sheet",
        imageUrl: "/idle.png",
        frameUrls: ["/idle.png"],
        frameWidth: 32,
        frameHeight: 32,
        columns: 1,
        rows: 1,
      },
    },
  ];
  if (record.mode === "character") record.character.animations = animations;
  if (record.mode === "object") record.object.animations = animations;
  return record;
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
    frameWidth: 48,
    frameHeight: 48,
    fps: 12,
    duration: 5,
    ...overrides,
  };
}
