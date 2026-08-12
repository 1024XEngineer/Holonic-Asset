import { beforeEach, describe, expect, it, vi } from "vitest";

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

import { generationApi, toCreateGenerationRequest } from "./generation.api";

beforeEach(() => {
  vi.clearAllMocks();
  mocks.core.create.mockResolvedValue({ generationRunId: 17 });
  mocks.core.list.mockResolvedValue({ items: [] });
  mocks.readFileAsDataUrl.mockResolvedValue("data:image/png;base64,reference");
});

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

    await expect(
      generationApi.enqueue({ projectId: "42", request }),
    ).resolves.toMatchObject({
      id: "17",
      projectId: "42",
      name: request.name,
      status: "pending",
    });
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

  it("rejects asset kinds that are outside the connected form scope", async () => {
    await expect(
      toCreateGenerationRequest(creationRequest({ kind: "audio" })),
    ).rejects.toThrow("supports Character and Object assets only");
  });

  it("rejects malformed canvas dimensions before enqueueing", async () => {
    await expect(
      toCreateGenerationRequest(creationRequest({ canvasSize: "large" })),
    ).rejects.toThrow("positive width × height");
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
