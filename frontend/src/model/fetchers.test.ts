import { afterEach, describe, expect, it, vi } from "vitest";

import { DataApiError } from "@/lib/data-api-error";

import {
  deleteEnvelope,
  getEnvelope,
  postEnvelope,
  putEnvelope,
} from "./fetchers";

afterEach(() => vi.unstubAllGlobals());

function response(body: unknown, init: ResponseInit = {}) {
  return new Response(body === undefined ? undefined : JSON.stringify(body), {
    status: 200,
    headers: { "Content-Type": "application/json" },
    ...init,
  });
}

describe("API fetchers", () => {
  it("serializes query primitives and repeated values", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(
        response({ code: 200, message: "ok", data: { id: 7 } }),
      );
    vi.stubGlobal("fetch", fetchMock);

    await expect(
      getEnvelope("/assets", {
        projectId: 12,
        active: true,
        tag: ["hero", "pixel art"],
        omitted: undefined,
        empty: null,
      }),
    ).resolves.toEqual({ id: 7 });

    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/assets?projectId=12&active=true&tag=hero&tag=pixel+art",
      {},
    );
  });

  it.each([
    ["POST", postEnvelope],
    ["PUT", putEnvelope],
    ["DELETE", deleteEnvelope],
  ] as const)("sends JSON bodies with %s", async (method, request) => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(response({ code: 200, message: "", data: "done" }));
    vi.stubGlobal("fetch", fetchMock);

    await expect(request("/resource", { id: 3 })).resolves.toBe("done");
    expect(fetchMock).toHaveBeenCalledWith("/api/v1/resource", {
      method,
      body: JSON.stringify({ id: 3 }),
      headers: { "Content-Type": "application/json" },
    });
  });

  it("does not add body headers when a body is omitted", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(response({ code: 200, message: "", data: null }));
    vi.stubGlobal("fetch", fetchMock);

    await postEnvelope("/resource");

    expect(fetchMock).toHaveBeenCalledWith("/api/v1/resource", {
      method: "POST",
    });
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

    await expect(getEnvelope("/resource")).rejects.toMatchObject({
      name: "DataApiError",
      code,
      message: `API request failed (${status}).`,
      details: { reason: "failed" },
    });
  });

  it("wraps network failures as unavailable errors", async () => {
    const failure = new Error("offline");
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(failure));

    await expect(getEnvelope("/resource")).rejects.toMatchObject({
      code: "UNAVAILABLE",
      details: failure,
    });
  });

  it("adds the current bearer token to API requests", async () => {
    vi.stubGlobal(
      "localStorage",
      createAuthStorage({
        accessToken: "signed-token",
        tokenType: "Bearer",
        expiresIn: 3_600,
        expiresAt: Date.now() + 3_600_000,
        user: { id: 7, username: "kay", email: "kay@example.com" },
      }),
    );
    const fetchMock = vi
      .fn()
      .mockResolvedValue(response({ code: 200, message: "", data: null }));
    vi.stubGlobal("fetch", fetchMock);

    await getEnvelope("/resource");

    expect(fetchMock).toHaveBeenCalledWith("/api/v1/resource", {
      headers: { Authorization: "Bearer signed-token" },
    });
  });

  it("maps unauthorized responses and clears the session", async () => {
    const storage = createAuthStorage({
      accessToken: "expired-token",
      tokenType: "Bearer",
      expiresIn: 3_600,
      expiresAt: Date.now() + 3_600_000,
      user: { id: 7, username: "kay", email: "kay@example.com" },
    });
    vi.stubGlobal("localStorage", storage);
    vi.stubGlobal(
      "fetch",
      vi
        .fn()
        .mockResolvedValue(response({ detail: "expired" }, { status: 401 })),
    );

    await expect(getEnvelope("/resource")).rejects.toMatchObject({
      code: "UNAUTHORIZED",
    });
    expect(storage.removeItem).toHaveBeenCalledWith("holonic-auth-session");
  });

  it("rejects unsuccessful API envelopes with their message", async () => {
    vi.stubGlobal(
      "fetch",
      vi
        .fn()
        .mockResolvedValueOnce(
          response({ code: 409, message: "Already exists", data: null }),
        )
        .mockResolvedValueOnce(
          response({ code: 500, message: "", data: null }),
        ),
    );

    await expect(getEnvelope("/resource")).rejects.toMatchObject({
      code: "CONFLICT",
      message: "Already exists",
    });
    await expect(getEnvelope("/resource")).rejects.toMatchObject({
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
      await getEnvelope("/resource");
      expect.unreachable();
    } catch (error) {
      expect(error).toBeInstanceOf(DataApiError);
      expect(error).toMatchObject({ details: "upstream failure" });
    }
  });
});

function createAuthStorage(session: unknown) {
  let value: string | null = JSON.stringify(session);
  return {
    getItem: vi.fn(() => value),
    setItem: vi.fn((_key: string, nextValue: string) => {
      value = nextValue;
    }),
    removeItem: vi.fn(() => {
      value = null;
    }),
  };
}
