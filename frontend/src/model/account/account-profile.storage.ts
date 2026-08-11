import { defaultAccountProfile } from "./mock";
import type { AccountProfile } from "./types";

const profileStorageKey = "holonic-account-profile";
export const accountProfileUpdatedEvent = "holonic-account-profile-updated";

export function readAccountProfile(): AccountProfile {
  const storedProfile = window.localStorage.getItem(profileStorageKey);
  if (!storedProfile) return defaultAccountProfile;

  try {
    const parsedProfile = JSON.parse(storedProfile) as Partial<AccountProfile>;
    return { ...defaultAccountProfile, ...parsedProfile };
  } catch {
    return defaultAccountProfile;
  }
}

export function saveAccountProfile(profile: AccountProfile) {
  window.localStorage.setItem(profileStorageKey, JSON.stringify(profile));
  window.dispatchEvent(new Event(accountProfileUpdatedEvent));
}
