import { initReactI18next, I18nextProvider } from "react-i18next";
import { createInstance, type i18n as I18nInstance } from "i18next";

import {
  defaultLanguage,
  defaultNamespace,
  resources,
  supportedLanguages,
} from "@/i18n/resources";

const testI18n = createInstance();
testI18n.use(initReactI18next).init({
  resources,
  lng: defaultLanguage,
  fallbackLng: defaultLanguage,
  supportedLngs: supportedLanguages,
  defaultNS: defaultNamespace,
  interpolation: { escapeValue: false },
});

export function withI18n(element: React.ReactNode): React.ReactElement {
  return <I18nextProvider i18n={testI18n}>{element}</I18nextProvider>;
}

export type TestI18n = I18nInstance;
