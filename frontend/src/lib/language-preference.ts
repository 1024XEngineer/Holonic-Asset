export const supportedLanguages = ["en-US", "zh-CN"] as const;

export type Language = (typeof supportedLanguages)[number];

const languageStorageKey = "holonic-language";

function isLanguage(value: string | null): value is Language {
  return value !== null && supportedLanguages.includes(value as Language);
}

export function readLanguagePreference(): Language {
  if (typeof window === "undefined") return "en-US";
  const stored = window.localStorage.getItem(languageStorageKey);
  if (isLanguage(stored)) return stored;
  return navigator.language.toLowerCase().startsWith("zh") ? "zh-CN" : "en-US";
}

export function saveLanguagePreference(language: Language) {
  window.localStorage.setItem(languageStorageKey, language);
  document.documentElement.lang = language;
}
