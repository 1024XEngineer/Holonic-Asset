import {
  defaultLanguage,
  isSupportedLanguage,
  type Language,
} from "./resources";

const languageStorageKey = "holonic-language";

export function resolveLanguagePreference(
  storedLanguage: string | null,
  requestedLanguages: readonly string[],
): Language {
  if (storedLanguage && isSupportedLanguage(storedLanguage)) {
    return storedLanguage;
  }

  for (const language of requestedLanguages) {
    const normalizedLanguage = language.toLowerCase();
    if (normalizedLanguage === "zh" || normalizedLanguage.startsWith("zh-")) {
      return "zh-CN";
    }
    if (normalizedLanguage === "en" || normalizedLanguage.startsWith("en-")) {
      return "en-US";
    }
  }

  return defaultLanguage;
}

export function readLanguagePreference(): Language {
  if (typeof window === "undefined") return defaultLanguage;

  let storedLanguage: string | null = null;
  try {
    storedLanguage = window.localStorage.getItem(languageStorageKey);
  } catch {
    // Language detection still works when storage is unavailable.
  }

  return resolveLanguagePreference(
    storedLanguage,
    navigator.languages.length > 0 ? navigator.languages : [navigator.language],
  );
}

export function saveLanguagePreference(language: Language): void {
  if (typeof window === "undefined") return;

  try {
    window.localStorage.setItem(languageStorageKey, language);
  } catch {
    // The active language remains valid even when persistence is unavailable.
  }
}
