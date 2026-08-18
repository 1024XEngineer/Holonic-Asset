import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  resolve: vi.fn(),
}));

vi.mock("./core-generation.api", () => ({
  coreGenerationApi: {
    resolveApplication: mocks.resolve,
  },
}));

import { resolveGenerationApplication } from "./generation-application.mutation";

beforeEach(() => {
  vi.clearAllMocks();
  mocks.resolve.mockResolvedValue({});
});

describe("resolveGenerationApplication", () => {
  it.each([true, false])(
    "only resolves the task when applied is %s",
    async (applied) => {
      await resolveGenerationApplication({
        projectId: "7",
        assetId: "9",
        runId: "12",
        applied,
      });

      expect(mocks.resolve).toHaveBeenCalledWith(12, applied);
    },
  );

  it("rejects a non-persisted generation run ID", async () => {
    await expect(
      resolveGenerationApplication({
        projectId: "7",
        assetId: "9",
        runId: "draft",
        applied: true,
      }),
    ).rejects.toThrow(
      "generation run requires a persisted Core API identifier",
    );
    expect(mocks.resolve).not.toHaveBeenCalled();
  });
});
