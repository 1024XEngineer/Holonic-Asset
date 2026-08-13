import { afterEach, describe, expect, it, vi } from "vitest";

import {
  AuthSessionPersistenceError,
  clearAuthSession,
  readAccessToken,
  readAuthSession,
  saveAuthSession,
  subscribeAuthSession,
} from "./auth-session.store";

afterEach(() => vi.unstubAllGlobals());

describe("auth session store", () => {
  it("stores login data with an absolute expiry and stable snapshot", () => {
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
    expect(readAuthSession()).toBe(session);
    expect(readAccessToken()).toBe("signed-token");
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

  it("clears sessions and notifies semantic subscribers", () => {
    const storage = createStorage();
    storage.removeItem.mockImplementation(() => {
      throw new DOMException("Storage is blocked", "SecurityError");
    });
    vi.stubGlobal("localStorage", storage);
    const listener = vi.fn();
    const unsubscribe = subscribeAuthSession(listener);

    clearAuthSession();

    expect(listener).toHaveBeenCalledOnce();
    unsubscribe();
  });

  it("reports storage write failures and unavailable storage", () => {
    const storage = createStorage();
    storage.setItem.mockImplementation(() => {
      throw new DOMException("Storage is blocked", "SecurityError");
    });
    vi.stubGlobal("localStorage", storage);

    expect(() => saveAuthSession(loginResponse())).toThrow(
      AuthSessionPersistenceError,
    );

    vi.stubGlobal("localStorage", undefined);
    expect(() => saveAuthSession(loginResponse())).toThrow(
      AuthSessionPersistenceError,
    );
  });

  it("synchronizes subscribers after a cross-tab storage change", () => {
    const storage = createStorage();
    const eventTarget = new EventTarget();
    vi.stubGlobal("localStorage", storage);
    vi.stubGlobal("window", eventTarget);
    const listener = vi.fn();
    const unsubscribe = subscribeAuthSession(listener);

    storage.setItem(
      "holonic-auth-session",
      JSON.stringify({
        ...loginResponse(),
        accessToken: "other-tab-token",
        expiresAt: Date.now() + 3_600_000,
      }),
    );
    eventTarget.dispatchEvent(
      Object.assign(new Event("storage"), { key: null }),
    );

    expect(listener).toHaveBeenCalledOnce();
    expect(readAccessToken()).toBe("other-tab-token");
    unsubscribe();
  });
});

function loginResponse() {
  return {
    accessToken: "signed-token",
    tokenType: "Bearer" as const,
    expiresIn: 3_600,
    user: { id: 7, username: "kay", email: "kay@example.com" },
  };
}

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
