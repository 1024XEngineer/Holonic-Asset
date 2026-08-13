import { createInstance } from "i18next";
import { initReactI18next } from "react-i18next";

import {
  readLanguagePreference,
  saveLanguagePreference,
} from "./language-preference";
import {
  defaultLanguage,
  defaultNamespace,
  isSupportedLanguage,
  resources,
  supportedLanguages,
  type Language,
} from "./resources";

export const i18n = createInstance();

let initialization: Promise<void> | undefined;

function applyDocumentLanguage(language: string): void {
  if (typeof document !== "undefined") {
    document.documentElement.lang = language;
  }
}

i18n.on("languageChanged", (language) => {
  if (!isSupportedLanguage(language)) return;
  saveLanguagePreference(language);
  applyDocumentLanguage(language);
});

export function initializeI18n(): Promise<void> {
  initialization ??= i18n
    .use(initReactI18next)
    .init({
      resources,
      lng: readLanguagePreference(),
      fallbackLng: defaultLanguage,
      supportedLngs: supportedLanguages,
      load: "currentOnly",
      defaultNS: defaultNamespace,
      returnEmptyString: false,
      interpolation: { escapeValue: false },
    })
    .then(() => undefined);

  return initialization.then(() => {
    applyDocumentLanguage(i18n.resolvedLanguage ?? defaultLanguage);
  });
}

export async function changeLanguage(language: Language): Promise<void> {
  await initializeI18n();
  await i18n.changeLanguage(language);
}

export default i18n;
