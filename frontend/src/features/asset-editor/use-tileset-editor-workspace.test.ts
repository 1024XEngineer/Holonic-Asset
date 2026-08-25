import { beforeEach, describe, expect, it, vi } from "vitest";

import type {
  AssetRecord,
  AssetWorkspaceData,
  GenerationTaskType,
  TileSetAssetContent,
} from "@/model";

const mocks = vi.hoisted(() => ({
  applicationMutation: {
    isPending: false,
    mutateAsync: vi.fn(),
  },
  candidateQuery: {
    data: undefined as
      | {
          kind: GenerationTaskType;
          result?: { content?: unknown };
          status: string;
        }
      | undefined,
  },
  coreCreate: vi.fn(),
  generationRuns: [] as Array<{
    id: string;
    name: string;
    prompt: string;
    status: string;
    error?: string;
  }>,
  rememberGenerationRunMetadata: vi.fn(),
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
  candidateRecordOverride: undefined as
    | ((record: AssetRecord, patch: TileSetAssetContent) => AssetRecord)
    | undefined,
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
  useTimeout: () => ({ schedule: (callback: () => void) => callback() }),
}));

vi.mock("@/model", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/model")>();
  return {
    ...actual,
    toCoreTilesetCandidateRecord: (
      ...args: Parameters<typeof actual.toCoreTilesetCandidateRecord>
    ) =>
      mocks.candidateRecordOverride?.(...args) ??
      actual.toCoreTilesetCandidateRecord(...args),
    coreGenerationApi: { create: mocks.coreCreate },
    rememberGenerationRunMetadata: mocks.rememberGenerationRunMetadata,
    useGenerationCandidateQuery: () => mocks.candidateQuery,
    useGenerationRunsQuery: () => ({ data: mocks.generationRuns }),
    useResolveGenerationApplicationMutation: () => mocks.applicationMutation,
  };
});

vi.mock("./state", () => ({ useEditorSession: () => mocks.session }));

import { useTilesetEditorWorkspace } from "./use-tileset-editor-workspace";

beforeEach(() => {
  vi.clearAllMocks();
  mocks.stateValues.length = 0;
  mocks.generationRuns = [];
  mocks.candidateQuery.data = undefined;
  mocks.applicationMutation.isPending = false;
  mocks.applicationMutation.mutateAsync.mockResolvedValue(undefined);
  mocks.candidateRecordOverride = undefined;
  mocks.coreCreate.mockResolvedValue({ generationRunId: 31 });
  mocks.session.save.mockResolvedValue({ status: "saved" });
  mocks.session.snapshot = snapshot(tilesetRecord());
});

describe("useTilesetEditorWorkspace", () => {
  it("submits an Item edit with only the managed creating reference", async () => {
    const editor = useTilesetEditorWorkspace({
      data: workspace(mocks.session.snapshot.record),
      onBack: vi.fn(),
    });
    if (!editor) return;

    await editor.onSubmit(
      {
        prompt: "Add moss",
        creatingReference: {
          fileName: "moss.png",
          mimeType: "image/png",
          objectKey: "uploads/moss.png",
          previewUrl: "https://cdn.example/moss.png",
        },
      },
      {
        kind: "item",
        itemId: "ground",
        label: "Ground",
        position: [0, 0],
        positions: [
          [0, 0],
          [1, 0],
        ],
      },
    );

    expect(mocks.coreCreate).toHaveBeenCalledWith(7, {
      assetId: 8,
      kind: "edit_tileset_item",
      creative_brief: "Add moss",
      parameters: {
        target: { position: { x: 0, y: 0 } },
        creating_reference: "uploads/moss.png",
      },
    });
    expect(mocks.rememberGenerationRunMetadata).toHaveBeenCalledWith("7", 31, {
      kind: "tileset",
      name: "Edit Asset",
      prompt: "Add moss",
      assetId: "8",
    });
  });

  it("previews and applies a partial candidate without saving", async () => {
    mocks.generationRuns = [
      {
        id: "ready",
        name: "Edit Asset",
        prompt: "Add cracks",
        status: "awaiting_application",
      },
    ];
    mocks.candidateQuery.data = {
      kind: "edit_tiles",
      status: "awaiting_application",
      result: {
        content: {
          items: [
            {
              name: "Ground",
              tiles: [{ position: { x: 1, y: 0 }, url: "/new-1.png" }],
            },
          ],
        },
      },
    };

    const editor = useTilesetEditorWorkspace({
      data: workspace(mocks.session.snapshot.record),
      onBack: vi.fn(),
    });

    expect(editor?.items[0]?.tileUrls).toEqual(["/old-0.png", "/new-1.png"]);
    expect(editor?.review).toEqual({
      items: [
        {
          itemId: "ground",
          currentItem: expect.objectContaining({
            id: "ground",
            tileUrls: ["/old-0.png", "/old-1.png"],
          }),
          candidateItem: expect.objectContaining({
            id: "ground",
            tileUrls: ["/old-0.png", "/new-1.png"],
          }),
        },
      ],
      isResolving: false,
    });
    editor?.onResolveReview(true);
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
        mode: "tileset",
        tileset: expect.objectContaining({
          items: [
            expect.objectContaining({
              tileUrls: ["/old-0.png", "/new-1.png"],
            }),
          ],
        }),
      }),
    });
    expect(mocks.session.save).not.toHaveBeenCalled();
  });

  it("denies a candidate without changing the editor record", async () => {
    mocks.generationRuns = [
      {
        id: "ready",
        name: "Edit Asset",
        prompt: "Add cracks",
        status: "awaiting_application",
      },
    ];
    mocks.candidateQuery.data = {
      kind: "edit_tiles",
      status: "awaiting_application",
      result: {
        content: {
          items: [
            {
              name: "Ground",
              tiles: [{ position: { x: 0, y: 0 }, url: "/new-0.png" }],
            },
          ],
        },
      },
    };
    const editor = useTilesetEditorWorkspace({
      data: workspace(mocks.session.snapshot.record),
      onBack: vi.fn(),
    });

    editor?.onResolveReview(false);
    await flushPromises();

    expect(mocks.applicationMutation.mutateAsync).toHaveBeenCalledWith(
      expect.objectContaining({ runId: "ready", applied: false }),
    );
    expect(mocks.session.dispatch).not.toHaveBeenCalled();
  });

  it("rejects edits when the asset identifiers are not persisted", async () => {
    const editor = useTilesetEditorWorkspace({
      data: workspace(mocks.session.snapshot.record, {
        id: "asset-temp",
        projectId: "project-temp",
      }),
      onBack: vi.fn(),
    });

    await expect(
      editor?.onSubmit(
        { prompt: "Add moss" },
        {
          kind: "item",
          itemId: "ground",
          label: "Ground",
          position: [0, 0],
          positions: [[0, 0]],
        },
      ),
    ).rejects.toThrow("persisted identifiers");
  });

  it("returns no editor for a non-tileset session", () => {
    mocks.session.snapshot = snapshot({
      mode: "uiset",
      prompt: "Menu",
      uiset: { components: [] },
    });

    expect(
      useTilesetEditorWorkspace({
        data: workspace(mocks.session.snapshot.record),
        onBack: vi.fn(),
      }),
    ).toBeNull();
  });

  it("omits reviews when a changed item is absent from the candidate", () => {
    mocks.generationRuns = [
      {
        id: "ready",
        name: "Edit Asset",
        prompt: "Add cracks",
        status: "awaiting_application",
      },
    ];
    mocks.candidateQuery.data = {
      kind: "edit_tiles",
      status: "awaiting_application",
      result: {
        content: {
          items: [
            {
              name: "Ground",
              tiles: [{ position: { x: 0, y: 0 }, url: "/new.png" }],
            },
          ],
        },
      },
    };
    mocks.candidateRecordOverride = () => {
      const record = tilesetRecord() as Extract<
        AssetRecord,
        { mode: "tileset" }
      >;
      return { ...record, tileset: { ...record.tileset, items: [] } };
    };

    const editor = useTilesetEditorWorkspace({
      data: workspace(mocks.session.snapshot.record),
      onBack: vi.fn(),
    });

    expect(editor?.review?.items).toEqual([]);
  });
});

function tilesetRecord(): AssetRecord {
  return {
    mode: "tileset",
    prompt: "Forest",
    tileset: {
      gridSize: 4,
      items: [
        {
          id: "ground",
          label: "Ground",
          tiles: [
            [0, 0],
            [1, 0],
          ],
          tileUrls: ["/old-0.png", "/old-1.png"],
        },
      ],
    },
  };
}

function snapshot(record: AssetRecord) {
  return {
    record,
    dirty: false,
    canUndo: false,
    canRedo: false,
    saveState: { phase: "idle" as const },
  };
}

function workspace(
  record: AssetRecord,
  identifiers: { id?: string; projectId?: string } = {},
): AssetWorkspaceData {
  return {
    projectName: "Project",
    asset: {
      id: identifiers.id ?? "8",
      projectId: identifiers.projectId ?? "7",
      kind: "tileset",
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
