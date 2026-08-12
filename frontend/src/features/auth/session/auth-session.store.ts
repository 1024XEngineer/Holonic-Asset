import { z } from "zod";

import type { LoginResponse } from "@/model/auth";

export type AuthSession = LoginResponse & {
  expiresAt: number;
};

export class AuthSessionPersistenceError extends Error {
  readonly storageError?: unknown;

  constructor(storageError?: unknown) {
    super("Unable to persist the authentication session.");
    this.name = "AuthSessionPersistenceError";
    this.storageError = storageError;
  }
}

const authSessionStorageKey = "holonic-auth-session";
const authSessionSchema = z.object({
  accessToken: z.string().min(1),
  tokenType: z.literal("Bearer"),
  expiresIn: z.number(),
  expiresAt: z.number(),
  user: z.object({
    id: z.number(),
    username: z.string(),
    email: z.string(),
  }),
}) satisfies z.ZodType<AuthSession>;

const listeners = new Set<() => void>();
let cachedStorageValue: string | null | undefined;
let cachedSession: AuthSession | null = null;
let stopStorageSync: (() => void) | undefined;

export function readAuthSession(): AuthSession | null {
  const storage = getStorage();
  if (!storage) return null;

  let storedSession: string | null;
  try {
    storedSession = storage.getItem(authSessionStorageKey);
  } catch {
    return null;
  }

  if (
    storedSession === cachedStorageValue &&
    (!cachedSession || cachedSession.expiresAt > Date.now())
  ) {
    return cachedSession;
  }

  if (!storedSession) {
    updateSnapshot(null, null);
    return null;
  }

  try {
    const result = authSessionSchema.safeParse(JSON.parse(storedSession));
    if (!result.success || result.data.expiresAt <= Date.now()) {
      removeStoredSession(storage);
      updateSnapshot(null, null);
      return null;
    }
    updateSnapshot(storedSession, result.data);
    return result.data;
  } catch {
    removeStoredSession(storage);
    updateSnapshot(null, null);
    return null;
  }
}

export function saveAuthSession(response: LoginResponse): AuthSession {
  const session: AuthSession = {
    ...response,
    expiresAt: Date.now() + response.expiresIn * 1_000,
  };
  const storage = getStorage();
  if (!storage) throw new AuthSessionPersistenceError();

  const storedSession = JSON.stringify(session);
  try {
    storage.setItem(authSessionStorageKey, storedSession);
  } catch (error) {
    throw new AuthSessionPersistenceError(error);
  }

  updateSnapshot(storedSession, session);
  notifyListeners();
  return session;
}

export function clearAuthSession() {
  const storage = getStorage();
  if (storage) removeStoredSession(storage);
  updateSnapshot(null, null);
  notifyListeners();
}

export function subscribeAuthSession(listener: () => void) {
  listeners.add(listener);
  if (listeners.size === 1) startStorageSync();

  return () => {
    listeners.delete(listener);
    if (listeners.size === 0) {
      stopStorageSync?.();
      stopStorageSync = undefined;
    }
  };
}

export function readAccessToken(): string | undefined {
  return readAuthSession()?.accessToken;
}

function updateSnapshot(
  storedValue: string | null,
  session: AuthSession | null,
) {
  cachedStorageValue = storedValue;
  cachedSession = session;
}

function removeStoredSession(storage: Storage) {
  try {
    storage.removeItem(authSessionStorageKey);
  } catch {
    // In-memory subscribers must still observe an invalidated session.
  }
}

function getStorage(): Storage | undefined {
  try {
    return typeof localStorage === "undefined" ? undefined : localStorage;
  } catch {
    return undefined;
  }
}

function startStorageSync() {
  if (typeof window === "undefined") return;

  const handleStorage = (event: StorageEvent) => {
    if (event.key !== null && event.key !== authSessionStorageKey) return;
    cachedStorageValue = undefined;
    readAuthSession();
    notifyListeners();
  };
  window.addEventListener("storage", handleStorage);
  stopStorageSync = () => window.removeEventListener("storage", handleStorage);
}

function notifyListeners() {
  listeners.forEach((listener) => listener());
}
