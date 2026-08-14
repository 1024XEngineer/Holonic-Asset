import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import type { CreationRequest } from "./types";

const mocks = vi.hoisted(() => ({
  core: {
    create: vi.fn(),
    list: vi.fn(),
  },
  readFileAsDataUrl: vi.fn(),
}));

vi.mock("./core-generation.api", () => ({ coreGenerationApi: mocks.core }));
vi.mock("@/lib/read-file-as-data-url", () => ({
  readFileAsDataUrl: mocks.readFileAsDataUrl,
}));

import {
  forgetGenerationRunMetadata,
  generationApi,
  pruneGenerationRequests,
  rememberGenerationRunMetadata,
  toCreateGenerationRequest,
} from "./generation.api";

beforeEach(() => {
  vi.stubGlobal("localStorage", createStorage());
  pruneGenerationRequests("42", []);
  vi.clearAllMocks();
  mocks.core.create.mockResolvedValue({ generationRunId: 17 });
  mocks.core.list.mockResolvedValue({ items: [] });
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

    pruneGenerationRequests("42", []);
    await expect(generationApi.listRuns("42")).resolves.toEqual([
      expect.objectContaining({
        id: "17",
        name: "New character",
        prompt: "",
        canvasSize: "32 × 32 px",
      }),
    ]);
  });

  it("does not prune asset metadata when the project queue is empty", async () => {
    rememberGenerationRunMetadata("42", 23, {
      kind: "character",
      name: "Walk left",
      prompt: "A relaxed looping walk",
      assetId: "7",
    });

    await generationApi.listRuns("42");
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

    await expect(generationApi.listRuns("42", "7")).resolves.toEqual([
      expect.objectContaining({ name: "Walk left", status: "processing" }),
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
  });

  it("retains metadata for failed runs while pruning runs no longer listed", async () => {
    const request = creationRequest();
    await generationApi.enqueue({ projectId: "42", request });

    mocks.core.list.mockResolvedValue({
      items: [
        {
          id: 17,
          projectId: 42,
          kind: "generate_character_prototype",
          status: "failed",
        },
      ],
    });
    pruneGenerationRequests("42", ["17"]);
    await expect(generationApi.listRuns("42")).resolves.toEqual([
      expect.objectContaining({
        id: "17",
        name: request.name,
        prompt: request.prompt,
        status: "failed",
      }),
    ]);

    pruneGenerationRequests("42", []);
    mocks.core.list.mockResolvedValue({ items: [] });
    await expect(generationApi.listRuns("42")).resolves.toEqual([]);
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

    await expect(generationApi.listRuns("42", "7")).resolves.toEqual([]);
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
    ).rejects.toThrow("supports Character and Object assets only");
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
