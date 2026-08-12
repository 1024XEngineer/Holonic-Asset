import { afterEach, describe, expect, it, vi } from "vitest";

import {
  clearAuthSession,
  readAccessToken,
  readAuthenticatedUserId,
  readAuthSession,
  saveAuthSession,
} from "./auth-session.storage";

afterEach(() => vi.unstubAllGlobals());

describe("auth session storage", () => {
  it("stores login data with an absolute expiry", () => {
    vi.spyOn(Date, "now").mockReturnValue(1_000);
    const storage = createStorage();
    vi.stubGlobal("localStorage", storage);

    const session = saveAuthSession({
      accessToken: "signed-token",
      tokenType: "Bearer",
      expiresIn: 3_600,
      user: { id: 7, username: "kay", email: "kay@example.com" },
    });

    expect(session.expiresAt).toBe(3_601_000);
    expect(readAuthSession()).toEqual(session);
    expect(readAccessToken()).toBe("signed-token");
    expect(readAuthenticatedUserId()).toBe(7);
  });

  it("removes expired and malformed sessions", () => {
    vi.spyOn(Date, "now").mockReturnValue(10_000);
    const storage = createStorage(
      JSON.stringify({
        accessToken: "expired",
        tokenType: "Bearer",
        expiresIn: 1,
        expiresAt: 9_999,
        user: { id: 7, username: "kay", email: "kay@example.com" },
      }),
    );
    vi.stubGlobal("localStorage", storage);

    expect(readAuthSession()).toBeNull();
    expect(storage.removeItem).toHaveBeenCalledOnce();

    storage.setItem("holonic-auth-session", "not-json");
    expect(readAuthSession()).toBeNull();
    expect(storage.removeItem).toHaveBeenCalledTimes(2);
  });

  it("clears sessions and rejects missing users", () => {
    const storage = createStorage();
    vi.stubGlobal("localStorage", storage);

    expect(() => readAuthenticatedUserId()).toThrow(
      "Authentication is required.",
    );
    clearAuthSession();
    expect(storage.removeItem).toHaveBeenCalledWith("holonic-auth-session");
  });
});

function createStorage(initialValue: string | null = null) {
  let value = initialValue;
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
