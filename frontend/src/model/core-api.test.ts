import { afterEach, describe, expect, it, vi } from "vitest";

import { coreAssetApi } from "./asset/library/core-asset.api";
import { coreGenerationApi } from "./generation/run/core-generation.api";
import { coreProjectApi } from "./project/core-project.api";
import { uploadApi } from "./upload/upload.api";
import { authApi } from "./auth/auth.api";

afterEach(() => vi.unstubAllGlobals());

describe("core API clients", () => {
  it("routes every operation through the expected endpoint and method", async () => {
    const fetchMock = vi.fn(
      async (_input: Request) =>
        new Response(JSON.stringify({ code: 200, message: "", data: {} })),
    );
    vi.stubGlobal("fetch", fetchMock);

    await authApi.login({ username: "kay", password: "secret" });

    await coreProjectApi.create({} as never);
    await coreProjectApi.generateReference({} as never);
    await coreProjectApi.list(7);
    await coreProjectApi.detail(8);
    await coreProjectApi.update({} as never);
    await coreProjectApi.delete({} as never);

    await coreAssetApi.list(7, { types: ["character"] });
    await coreAssetApi.list(7);
    await coreAssetApi.detail(9);
    await coreAssetApi.records(9);
    await coreAssetApi.record({} as never);
    await coreAssetApi.copy({} as never);
    await coreAssetApi.rollback({} as never);
    await coreAssetApi.update({} as never);
    await coreAssetApi.delete({} as never);

    await coreGenerationApi.create(7, {} as never);
    await coreGenerationApi.list(7, { status: "active" });
    await coreGenerationApi.detail(10);
    await coreGenerationApi.cancel(10);
    await uploadApi.createTarget({} as never);

    expect(fetchMock).toHaveBeenCalledTimes(21);
    expect(fetchMock.mock.calls.map(([request]) => request.url)).toEqual(
      [
        "/auth/login",
        "/project/create",
        "/project/reference/generate",
        "/project/list?userID=7",
        "/project/detail?projectID=8",
        "/project/update",
        "/project/delete",
        "/projects/7/assets?types=character",
        "/projects/7/assets",
        "/asset/9",
        "/asset/9/records",
        "/asset/save",
        "/asset/copy",
        "/asset/rollback",
        "/asset/update",
        "/asset/delete",
        "/projects/7/generation-runs",
        "/projects/7/generation-runs?status=active",
        "/generation-runs/10",
        "/generation-runs/10/cancel",
        "/uploads",
      ].map((path) => `http://localhost/api/v1${path}`),
    );
    expect(fetchMock.mock.calls.map(([request]) => request.method)).toEqual([
      "POST",
      "POST",
      "POST",
      "GET",
      "GET",
      "PUT",
      "DELETE",
      "GET",
      "GET",
      "GET",
      "GET",
      "POST",
      "POST",
      "POST",
      "PUT",
      "DELETE",
      "POST",
      "GET",
      "GET",
      "POST",
      "POST",
    ]);
    expect(fetchMock.mock.calls[7]?.[0].cache).toBe("no-store");
    expect(fetchMock.mock.calls[8]?.[0].cache).toBe("no-store");
    expect(fetchMock.mock.calls[9]?.[0].cache).toBe("no-store");
  });
});
