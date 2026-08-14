import { afterEach, describe, expect, it, vi } from "vitest";

import { DataApiError } from "@/lib/data-api-error";

import {
  configureCoreApiAuth,
  coreApiClient,
  createCoreApiClients,
  publicCoreApiClient,
  unwrapApiResponse,
} from "./fetchers";

afterEach(() => {
  configureCoreApiAuth({
    getAccessToken: () => undefined,
    onUnauthorized: () => undefined,
  });
  vi.unstubAllGlobals();
});

function response(body: unknown, init: ResponseInit = {}) {
  return new Response(body === undefined ? undefined : JSON.stringify(body), {
    status: 200,
    headers: { "Content-Type": "application/json" },
    ...init,
  });
}

describe("core API client", () => {
  it("uses the injected API base URL", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(response({ code: 200, message: "ok", data: {} }));
    const clients = createCoreApiClients({
      baseUrl: "https://api.example.test/v2",
      fetch: fetchMock,
    });

    await clients.public.POST("/auth/login", {
      body: { username: "kay", password: "secret" },
    });

    expect(requestFrom(fetchMock).url).toBe(
      "https://api.example.test/v2/auth/login",
    );
  });

  it("serializes typed path, query, and repeated array parameters", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(
        response({ code: 200, message: "ok", data: { assets: [] } }),
      );
    vi.stubGlobal("fetch", fetchMock);

    const result = await coreApiClient.GET("/projects/{project_id}/assets", {
      params: {
        path: { project_id: 12 },
        query: {
          query: "pixel art",
          tags: ["hero", "featured"],
          types: ["character", "object"],
        },
      },
    });

    expect(unwrapApiResponse(result)).toEqual({ assets: [] });
    expect(requestFrom(fetchMock).url).toBe(
      "http://localhost/api/v1/projects/12/assets?query=pixel%20art&tags=hero&tags=featured&types=character&types=object",
    );
  });

  it("uses native Headers and serializes JSON request bodies", async () => {
    const fetchMock = vi
      .fn()
      .mockImplementation(() =>
        Promise.resolve(response({ code: 200, message: "", data: {} })),
      );
    vi.stubGlobal("fetch", fetchMock);

    await publicCoreApiClient.POST("/auth/login", {
      body: { username: "kay", password: "secret" },
      headers: { "X-Request-ID": "login-request" },
    });

    const request = requestFrom(fetchMock);
    expect(request.method).toBe("POST");
    expect(request.headers).toBeInstanceOf(Headers);
    expect(request.headers.get("Content-Type")).toBe("application/json");
    expect(request.headers.get("x-request-id")).toBe("login-request");
    expect(await request.text()).toBe(
      JSON.stringify({ username: "kay", password: "secret" }),
    );
  });

  it("does not add content headers when a body is omitted", async () => {
    const fetchMock = vi
      .fn()
      .mockImplementation(() =>
        Promise.resolve(response({ code: 200, message: "", data: {} })),
      );
    vi.stubGlobal("fetch", fetchMock);

    await coreApiClient.POST("/generation-runs/{run_id}/cancel", {
      params: { path: { run_id: 3 } },
    });

    expect(requestFrom(fetchMock).headers.has("Content-Type")).toBe(false);
  });

  it.each([
    [400, "BAD_REQUEST"],
    [422, "BAD_REQUEST"],
    [404, "NOT_FOUND"],
    [409, "CONFLICT"],
    [500, "UNKNOWN"],
  ] as const)("maps HTTP %s to %s", async (status, code) => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(response({ reason: "failed" }, { status })),
    );

    await expect(listProjects()).rejects.toMatchObject({
      name: "DataApiError",
      code,
      message: `API request failed (${status}).`,
      details: { reason: "failed" },
    });
  });

  it("wraps network failures as unavailable errors", async () => {
    const failure = new Error("offline");
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(failure));

    await expect(listProjects()).rejects.toMatchObject({
      code: "UNAVAILABLE",
      details: failure,
    });
  });

  it("adds the current bearer token with a case-insensitive override", async () => {
    configureCoreApiAuth({
      getAccessToken: () => "signed-token",
      onUnauthorized: () => undefined,
    });
    const fetchMock = vi
      .fn()
      .mockImplementation(() =>
        Promise.resolve(response({ code: 200, message: "", data: {} })),
      );
    vi.stubGlobal("fetch", fetchMock);

    await listProjects();
    expect(requestFrom(fetchMock).headers.get("Authorization")).toBe(
      "Bearer signed-token",
    );

    await coreApiClient.GET("/project/list", {
      params: { query: { userID: 7 } },
      headers: { authorization: "Bearer explicit-token" },
    });
    expect(requestFrom(fetchMock, 1).headers.get("Authorization")).toBe(
      "Bearer explicit-token",
    );
  });

  it("does not attach a bearer token to public requests", async () => {
    configureCoreApiAuth({
      getAccessToken: () => "old-token",
      onUnauthorized: () => undefined,
    });
    const fetchMock = vi
      .fn()
      .mockResolvedValue(response({ code: 200, message: "", data: {} }));
    vi.stubGlobal("fetch", fetchMock);

    await publicCoreApiClient.POST("/auth/login", {
      body: { username: "kay", password: "secret" },
    });

    expect(requestFrom(fetchMock).headers.has("Authorization")).toBe(false);
  });

  it("clears the session after an unauthorized response", async () => {
    const onUnauthorized = vi.fn();
    configureCoreApiAuth({
      getAccessToken: () => "expired-token",
      onUnauthorized,
    });
    vi.stubGlobal(
      "fetch",
      vi
        .fn()
        .mockResolvedValue(response({ detail: "expired" }, { status: 401 })),
    );

    await expect(listProjects()).rejects.toMatchObject({
      code: "UNAUTHORIZED",
    });
    expect(onUnauthorized).toHaveBeenCalledOnce();
  });

  it("rejects invalid and unsuccessful API envelopes", async () => {
    vi.stubGlobal(
      "fetch",
      vi
        .fn()
        .mockResolvedValueOnce(response({ unexpected: true }))
        .mockResolvedValueOnce(
          response({ code: 409, message: "Already exists", data: null }),
        )
        .mockResolvedValueOnce(
          response({ code: 500, message: "", data: null }),
        ),
    );

    await expect(listProjects()).rejects.toMatchObject({
      code: "UNKNOWN",
      message: "Invalid API response.",
    });
    await expect(listProjects()).rejects.toMatchObject({
      code: "CONFLICT",
      message: "Already exists",
    });
    await expect(listProjects()).rejects.toMatchObject({
      code: "UNKNOWN",
      message: "Request failed",
    });
  });

  it("preserves plain-text error bodies", async () => {
    vi.stubGlobal(
      "fetch",
      vi
        .fn()
        .mockResolvedValue(new Response("upstream failure", { status: 502 })),
    );

    try {
      await listProjects();
      expect.unreachable();
    } catch (error) {
      expect(error).toBeInstanceOf(DataApiError);
      expect(error).toMatchObject({ details: "upstream failure" });
    }
  });
});

async function listProjects() {
  return unwrapApiResponse(
    await coreApiClient.GET("/project/list", {
      params: { query: { userID: 7 } },
    }),
  );
}

function requestFrom(fetchMock: ReturnType<typeof vi.fn>, callIndex = 0) {
  return fetchMock.mock.calls[callIndex][0] as Request;
}
