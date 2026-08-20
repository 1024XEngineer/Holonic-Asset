import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import type { CreationRequest } from "./types";

const mocks = vi.hoisted(() => ({
  core: {
    create: vi.fn(),
    list: vi.fn(),
  },
  assetDetail: vi.fn(),
  readFileAsDataUrl: vi.fn(),
}));

vi.mock("./core-generation.api", () => ({ coreGenerationApi: mocks.core }));
vi.mock("../../asset/library/core-asset.api", () => ({
  coreAssetApi: { detail: mocks.assetDetail },
}));
vi.mock("@/lib/read-file-as-data-url", () => ({
  readFileAsDataUrl: mocks.readFileAsDataUrl,
}));

import {
  forgetGenerationRunMetadata,
  generationApi,
  rememberGenerationRunMetadata,
  toCreateGenerationRequest,
} from "./generation.api";

beforeEach(() => {
  vi.stubGlobal("localStorage", createStorage());
  forgetGenerationRunMetadata("42", ["17", "23"]);
  vi.clearAllMocks();
  mocks.core.create.mockResolvedValue({ generationRunId: 17 });
  mocks.core.list.mockResolvedValue({ items: [] });
  mocks.assetDetail.mockResolvedValue({ type: "character" });
  mocks.readFileAsDataUrl.mockResolvedValue("data:image/png;base64,reference");
});

afterEach(() => vi.unstubAllGlobals());

describe("generationApi", () => {
  it.each([
    ["character", "generate_character_prototype"],
    ["object", "generate_object_prototype"],
  ] as const)(
    "maps a %s form request to the Core API",
    async (kind, taskKind) => {
      const reference = new File(["image"], "reference.png", {
        type: "image/png",
      });
      const request = creationRequest({ kind, reference });

      await expect(toCreateGenerationRequest(request)).resolves.toEqual({
        kind: taskKind,
        creative_brief: "A moonlit orchard keeper",
        parameters: {
          asset_name: "Orchard Keeper",
          dimensions: { width: 48, height: 64 },
          perspective: "Isometric",
          reference: "data:image/png;base64,reference",
        },
      });
      expect(mocks.readFileAsDataUrl).toHaveBeenCalledWith(reference);
    },
  );

  it("maps a tileset form request to the Core API", async () => {
    const request = creationRequest({
      kind: "tileset",
      canvasSize: "16 × 16 px",
      perspective: undefined,
      tiles: [
        {
          name: "  Grass edge  ",
          description: "  A seamless grass edge  ",
          shape: [
            [0, 0],
            [1, 0],
          ],
        },
        {
          name: "Dirt",
          description: "A dirt tile",
          shape: [[0, 0]],
        },
      ],
    });

    await expect(toCreateGenerationRequest(request)).resolves.toEqual({
      kind: "generate_tileset",
      creative_brief: "A moonlit orchard keeper",
      parameters: {
        asset_name: "Orchard Keeper",
        dimensions: {
          tileSize: { width: 16, height: 16 },
          tileAmount: { columns: 16, rows: 16 },
        },
        items: [
          {
            name: "Grass edge",
            description: "A seamless grass edge",
            shape: [
              [0, 0],
              [1, 0],
            ],
          },
          {
            name: "Dirt",
            description: "A dirt tile",
            shape: [[0, 0]],
          },
        ],
      },
    });
  });

  it("maps a scenery form request to the Core API", async () => {
    const reference = new File(["image"], "reference.png", {
      type: "image/png",
    });
    const request = creationRequest({
      kind: "scenery",
      canvasSize: "stale display value",
      dimensions: { width: 1792, height: 768 },
      reference,
    });

    await expect(toCreateGenerationRequest(request)).resolves.toEqual({
      kind: "generate_scenery",
      creative_brief: "A moonlit orchard keeper",
      parameters: {
        asset_name: "Orchard Keeper",
        dimensions: { width: 1792, height: 768 },
        reference: "data:image/png;base64,reference",
      },
    });
    expect(mocks.readFileAsDataUrl).toHaveBeenCalledWith(reference);
  });

  it("resolves an awaiting animation edit to the owning asset kind", async () => {
    mocks.assetDetail.mockResolvedValue({ type: "object" });
    mocks.core.list.mockResolvedValue({
      items: [
        {
          id: 26,
          projectId: 42,
          assetId: 18,
          kind: "edit_animation",
          status: "awaiting_application",
        },
      ],
    });

    await expect(generationApi.listRuns("42", "18")).resolves.toEqual([
      expect.objectContaining({
        id: "26",
        kind: "object",
        status: "awaiting_application",
      }),
    ]);
    expect(mocks.assetDetail).toHaveBeenCalledWith(18);
  });

  it("resolves an awaiting frame edit to the owning asset kind", async () => {
    mocks.assetDetail.mockResolvedValue({ type: "object" });
    mocks.core.list.mockResolvedValue({
      items: [
        {
          id: 27,
          projectId: 42,
          assetId: 19,
          kind: "edit_frames",
          status: "awaiting_application",
        },
      ],
    });

    await expect(generationApi.listRuns("42", "19")).resolves.toEqual([
      expect.objectContaining({
        id: "27",
        kind: "object",
        status: "awaiting_application",
      }),
    ]);
    expect(mocks.assetDetail).toHaveBeenCalledWith(19);
  });

  it("creates and lists remote generation runs while preserving form metadata", async () => {
    const request = creationRequest();

    const run = await generationApi.enqueue({ projectId: "42", request });
    expect(run).toMatchObject({
      id: "17",
      projectId: "42",
      name: request.name,
      status: "pending",
    });
    expect(run).not.toHaveProperty("reference");
    expect(mocks.core.create).toHaveBeenCalledWith(
      42,
      expect.objectContaining({ kind: "generate_character_prototype" }),
    );

    mocks.core.list.mockResolvedValue({
      items: [
        {
          id: 17,
          projectId: 42,
          kind: "generate_character_prototype",
          status: "processing",
        },
      ],
    });
    await expect(generationApi.listRuns("42")).resolves.toEqual([
      expect.objectContaining({
        id: "17",
        name: request.name,
        prompt: request.prompt,
        status: "processing",
      }),
    ]);
    expect(mocks.core.list).toHaveBeenCalledWith(42, { status: "active" });

    forgetGenerationRunMetadata("42", ["17"]);
    await expect(generationApi.listRuns("42")).resolves.toEqual([
      expect.objectContaining({
        id: "17",
        name: "New character",
        prompt: "",
        canvasSize: "32 × 32 px",
      }),
    ]);
  });

  it("lists a scenery generation run without local metadata", async () => {
    mocks.core.list.mockResolvedValue({
      items: [
        {
          id: 31,
          projectId: 42,
          kind: "generate_scenery",
          status: "processing",
        },
      ],
    });

    await expect(generationApi.listRuns("42")).resolves.toEqual([
      expect.objectContaining({
        id: "31",
        kind: "scenery",
        name: "New scenery",
        status: "processing",
      }),
    ]);
  });

  it("restores generation metadata from browser storage", async () => {
    const request = creationRequest();
    await generationApi.enqueue({ projectId: "42", request });

    vi.resetModules();
    const reloaded = await import("./generation.api");
    mocks.core.list.mockResolvedValue({
      items: [
        {
          id: 17,
          projectId: 42,
          kind: "generate_character_prototype",
          status: "processing",
        },
      ],
    });

    await expect(reloaded.generationApi.listRuns("42")).resolves.toEqual([
      expect.objectContaining({ name: request.name, prompt: request.prompt }),
    ]);
  });

  it("includes queued animation runs when their form metadata is available", async () => {
    rememberGenerationRunMetadata("42", 23, {
      kind: "character",
      name: "Walk left",
      prompt: "A relaxed looping walk",
    });
    mocks.core.list.mockResolvedValue({
      items: [
        {
          id: 23,
          projectId: 42,
          assetId: 7,
          kind: "generate_animation",
          status: "pending",
        },
      ],
    });

    await expect(generationApi.listRuns("42", "7")).resolves.toEqual([
      expect.objectContaining({
        id: "23",
        kind: "character",
        name: "Walk left",
        prompt: "A relaxed looping walk",
        status: "pending",
      }),
    ]);
    expect(mocks.core.list).toHaveBeenCalledWith(42, {
      status: "active",
      assetId: 7,
    });
    expect(mocks.assetDetail).not.toHaveBeenCalled();
  });

  it("restores an animation kind from the Core asset without local metadata", async () => {
    mocks.assetDetail.mockResolvedValue({ type: "object" });
    mocks.core.list.mockResolvedValue({
      items: [
        {
          id: 23,
          projectId: 42,
          assetId: 8,
          kind: "generate_animation",
          status: "processing",
        },
      ],
    });

    await expect(generationApi.listRuns("42")).resolves.toEqual([
      expect.objectContaining({
        id: "23",
        kind: "object",
        name: "New object",
        status: "processing",
      }),
    ]);
    expect(mocks.assetDetail).toHaveBeenCalledWith(8);
  });

  it("keeps awaiting animation runs visible without local metadata", async () => {
    mocks.core.list.mockResolvedValue({
      items: [
        {
          id: 24,
          projectId: 42,
          kind: "generate_animation",
          status: "awaiting_application",
        },
      ],
    });

    await expect(generationApi.listRuns("42")).resolves.toEqual([
      expect.objectContaining({
        id: "24",
        kind: "character",
        status: "awaiting_application",
      }),
    ]);
    expect(mocks.assetDetail).not.toHaveBeenCalled();
  });

  it("forgets animation metadata after a settled run is reconciled", async () => {
    rememberGenerationRunMetadata("42", 23, {
      kind: "character",
      name: "Walk left",
      prompt: "A relaxed looping walk",
      assetId: "7",
    });
    mocks.core.list.mockResolvedValue({
      items: [
        {
          id: 23,
          projectId: 42,
          assetId: 7,
          kind: "generate_animation",
          status: "processing",
        },
      ],
    });
    await expect(generationApi.listRuns("42", "7")).resolves.toHaveLength(1);

    forgetGenerationRunMetadata("42", ["23"]);

    await expect(generationApi.listRuns("42", "7")).resolves.toEqual([
      expect.objectContaining({
        id: "23",
        kind: "character",
        name: "New character",
      }),
    ]);
  });

  it("rejects projects that have not been persisted by the Core API", async () => {
    await expect(
      generationApi.enqueue({
        projectId: "moonlit-orchard",
        request: creationRequest(),
      }),
    ).rejects.toThrow("persisted Core API project");
    await expect(generationApi.listRuns("moonlit-orchard")).rejects.toThrow(
      "persisted Core API project",
    );
    expect(mocks.core.create).not.toHaveBeenCalled();
    expect(mocks.core.list).not.toHaveBeenCalled();
  });

  it("rejects assets that have not been persisted by the Core API", async () => {
    await expect(generationApi.listRuns("42", "draft")).rejects.toThrow(
      "persisted Core API asset",
    );
    expect(mocks.core.list).not.toHaveBeenCalled();
  });

  it("rejects asset kinds that are outside the connected form scope", async () => {
    await expect(
      toCreateGenerationRequest(creationRequest({ kind: "audio" })),
    ).rejects.toThrow(
      "supports Character, Object, Scenery, and Tileset assets only",
    );
  });

  it("rejects a tileset without items", async () => {
    await expect(
      toCreateGenerationRequest(
        creationRequest({
          kind: "tileset",
          canvasSize: "16 × 16 px",
          perspective: undefined,
          tiles: [],
        }),
      ),
    ).rejects.toThrow("At least one tileset item is required");
  });

  it("rejects malformed canvas dimensions before enqueueing", async () => {
    await expect(
      toCreateGenerationRequest(creationRequest({ canvasSize: "large" })),
    ).rejects.toThrow("Canvas size must use a positive width × height value.");
  });
});

function creationRequest(
  overrides: Partial<CreationRequest> = {},
): CreationRequest {
  return {
    kind: "character",
    name: "Orchard Keeper",
    prompt: "A moonlit orchard keeper",
    canvasSize: "48 × 64 px",
    perspective: "Isometric",
    ...overrides,
  };
}

function createStorage() {
  const values = new Map<string, string>();
  return {
    getItem: vi.fn((key: string) => values.get(key) ?? null),
    setItem: vi.fn((key: string, value: string) => values.set(key, value)),
    removeItem: vi.fn((key: string) => values.delete(key)),
  };
}
