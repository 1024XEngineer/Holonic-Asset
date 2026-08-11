import { afterEach, describe, expect, it, vi } from "vitest";

import { coreAssetApi } from "./asset/library/core-asset.api";
import { coreGenerationApi } from "./generation/run/core-generation.api";
import { coreProjectApi } from "./project/core-project.api";
import { uploadApi } from "./upload/upload.api";

afterEach(() => vi.unstubAllGlobals());

describe("core API clients", () => {
  it("routes every operation through the expected endpoint and method", async () => {
    const fetchMock = vi.fn(
      async (_input: RequestInfo | URL, _init?: RequestInit) =>
        new Response(JSON.stringify({ code: 200, message: "", data: {} })),
    );
    vi.stubGlobal("fetch", fetchMock);

    await coreProjectApi.create({} as never);
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

    expect(fetchMock).toHaveBeenCalledTimes(19);
    expect(fetchMock.mock.calls.map(([url]) => url)).toEqual([
      "/api/v1/project/create",
      "/api/v1/project/list?userID=7",
      "/api/v1/project/detail?projectID=8",
      "/api/v1/project/update",
      "/api/v1/project/delete",
      "/api/v1/projects/7/assets?types=character",
      "/api/v1/projects/7/assets",
      "/api/v1/asset/9",
      "/api/v1/asset/9/records",
      "/api/v1/asset/save",
      "/api/v1/asset/copy",
      "/api/v1/asset/rollback",
      "/api/v1/asset/update",
      "/api/v1/asset/delete",
      "/api/v1/projects/7/generation-runs",
      "/api/v1/projects/7/generation-runs?status=active",
      "/api/v1/generation-runs/10",
      "/api/v1/generation-runs/10/cancel",
      "/api/v1/uploads",
    ]);
    expect(
      fetchMock.mock.calls.map(([, init]) => init?.method ?? "GET"),
    ).toEqual([
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
  });
});
