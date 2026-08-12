import { defaultAccountProfile } from "./account-profile";
import type { AccountProfile } from "./account-profile";

const accountProfileStorageKey = "holonic-account-profile";
const listeners = new Set<() => void>();
let cachedStorageValue: string | null | undefined;
let cachedProfile = defaultAccountProfile;
let stopStorageSync: (() => void) | undefined;

export function readAccountProfile(): AccountProfile {
  const storage = getStorage();
  if (!storage) return defaultAccountProfile;

  let storedProfile: string | null;
  try {
    storedProfile = storage.getItem(accountProfileStorageKey);
  } catch {
    return defaultAccountProfile;
  }

  if (storedProfile === cachedStorageValue) return cachedProfile;
  if (!storedProfile) return updateSnapshot(null, defaultAccountProfile);

  try {
    const parsedProfile = JSON.parse(storedProfile) as Partial<AccountProfile>;
    return updateSnapshot(storedProfile, {
      ...defaultAccountProfile,
      ...parsedProfile,
    });
  } catch {
    return updateSnapshot(storedProfile, defaultAccountProfile);
  }
}

export function saveAccountProfile(profile: AccountProfile) {
  const storedProfile = JSON.stringify(profile);
  window.localStorage.setItem(accountProfileStorageKey, storedProfile);
  updateSnapshot(storedProfile, profile);
  notifyListeners();
}

export function subscribeAccountProfile(listener: () => void) {
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

function updateSnapshot(storedValue: string | null, profile: AccountProfile) {
  cachedStorageValue = storedValue;
  cachedProfile = profile;
  return profile;
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
    if (event.key !== null && event.key !== accountProfileStorageKey) return;
    cachedStorageValue = undefined;
    readAccountProfile();
    notifyListeners();
  };
  window.addEventListener("storage", handleStorage);
  stopStorageSync = () => window.removeEventListener("storage", handleStorage);
}

function notifyListeners() {
  listeners.forEach((listener) => listener());
}
