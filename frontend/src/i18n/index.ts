import i18n from "i18next";
import { initReactI18next } from "react-i18next";
import {
  readLanguagePreference,
  saveLanguagePreference,
} from "@/lib/language-preference";
import enCommon from "./locales/en-US/common.json";
import enNavigation from "./locales/en-US/navigation.json";
import enSettings from "./locales/en-US/settings.json";
import enStatus from "./locales/en-US/status.json";
import enWorkspace from "./locales/en-US/workspace.json";
import enDocs from "./locales/en-US/docs.json";
import zhCommon from "./locales/zh-CN/common.json";
import zhNavigation from "./locales/zh-CN/navigation.json";
import zhSettings from "./locales/zh-CN/settings.json";
import zhStatus from "./locales/zh-CN/status.json";
import zhWorkspace from "./locales/zh-CN/workspace.json";
import zhDocs from "./locales/zh-CN/docs.json";

const resources = {
  "en-US": {
    common: enCommon,
    navigation: enNavigation,
    settings: enSettings,
    status: enStatus,
    workspace: enWorkspace,
    docs: enDocs,
  },
  "zh-CN": {
    common: zhCommon,
    navigation: zhNavigation,
    settings: zhSettings,
    status: zhStatus,
    workspace: zhWorkspace,
    docs: zhDocs,
  },
} as const;

void i18n.use(initReactI18next).init({
  resources,
  lng: readLanguagePreference(),
  fallbackLng: "en-US",
  defaultNS: "common",
  interpolation: { escapeValue: false },
});

document.documentElement.lang = i18n.language;

export async function changeLanguage(language: keyof typeof resources) {
  await i18n.changeLanguage(language);
  saveLanguagePreference(language);
}

export { i18n };
export default i18n;
