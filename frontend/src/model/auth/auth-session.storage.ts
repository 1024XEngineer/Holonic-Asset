import type { LoginResponse } from "./auth.contract";

export type AuthSession = LoginResponse & {
  expiresAt: number;
};

const authSessionStorageKey = "holonic-auth-session";
export const authSessionUpdatedEvent = "holonic-auth-session-updated";

export function readAuthSession(): AuthSession | null {
  const storage = getStorage();
  if (!storage) return null;

  let storedSession: string | null;
  try {
    storedSession = storage.getItem(authSessionStorageKey);
  } catch {
    return null;
  }
  if (!storedSession) return null;

  try {
    const session = JSON.parse(storedSession) as unknown;
    if (!isAuthSession(session) || session.expiresAt <= Date.now()) {
      storage.removeItem(authSessionStorageKey);
      return null;
    }
    return session;
  } catch {
    try {
      storage.removeItem(authSessionStorageKey);
    } catch {
      // Storage can be blocked independently for reads and writes.
    }
    return null;
  }
}

export function saveAuthSession(response: LoginResponse): AuthSession {
  const session: AuthSession = {
    ...response,
    expiresAt: Date.now() + response.expiresIn * 1_000,
  };
  const storage = getStorage();
  if (!storage) return session;

  storage.setItem(authSessionStorageKey, JSON.stringify(session));
  dispatchAuthSessionUpdate();
  return session;
}

export function clearAuthSession() {
  const storage = getStorage();
  if (!storage) return;

  try {
    storage.removeItem(authSessionStorageKey);
  } catch {
    return;
  }
  dispatchAuthSessionUpdate();
}

export function readAccessToken(): string | undefined {
  return readAuthSession()?.accessToken;
}

export function readAuthenticatedUserId(): number {
  const userId = readAuthSession()?.user.id;
  if (userId === undefined) throw new Error("Authentication is required.");
  return userId;
}

function isAuthSession(value: unknown): value is AuthSession {
  if (!value || typeof value !== "object") return false;

  const session = value as Partial<AuthSession>;
  const user = session.user as Partial<AuthSession["user"]> | undefined;
  return (
    typeof session.accessToken === "string" &&
    session.accessToken.length > 0 &&
    session.tokenType === "Bearer" &&
    typeof session.expiresIn === "number" &&
    typeof session.expiresAt === "number" &&
    typeof user?.id === "number" &&
    typeof user.username === "string" &&
    typeof user.email === "string"
  );
}

function getStorage(): Storage | undefined {
  try {
    return typeof localStorage === "undefined" ? undefined : localStorage;
  } catch {
    return undefined;
  }
}

function dispatchAuthSessionUpdate() {
  if (typeof window !== "undefined") {
    window.dispatchEvent(new Event(authSessionUpdatedEvent));
  }
}
