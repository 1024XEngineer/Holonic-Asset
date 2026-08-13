import enAssets from "./locales/en-US/assets.json";
import enAuth from "./locales/en-US/auth.json";
import enCommon from "./locales/en-US/common.json";
import enDocs from "./locales/en-US/docs.json";
import enEditor from "./locales/en-US/editor.json";
import enGeneration from "./locales/en-US/generation.json";
import enHome from "./locales/en-US/home.json";
import enNavigation from "./locales/en-US/navigation.json";
import enProjects from "./locales/en-US/projects.json";
import enSettings from "./locales/en-US/settings.json";
import enStatus from "./locales/en-US/status.json";
import zhAssets from "./locales/zh-CN/assets.json";
import zhAuth from "./locales/zh-CN/auth.json";
import zhCommon from "./locales/zh-CN/common.json";
import zhDocs from "./locales/zh-CN/docs.json";
import zhEditor from "./locales/zh-CN/editor.json";
import zhGeneration from "./locales/zh-CN/generation.json";
import zhHome from "./locales/zh-CN/home.json";
import zhNavigation from "./locales/zh-CN/navigation.json";
import zhProjects from "./locales/zh-CN/projects.json";
import zhSettings from "./locales/zh-CN/settings.json";
import zhStatus from "./locales/zh-CN/status.json";

export const defaultLanguage = "en-US";
export const defaultNamespace = "common";

export const languages = [
  { code: "en-US", name: "English" },
  { code: "zh-CN", name: "简体中文" },
] as const;

export type Language = (typeof languages)[number]["code"];

export const supportedLanguages = languages.map(({ code }) => code);

type TranslationShape<T> = {
  [Key in keyof T]: T[Key] extends string
    ? string
    : T[Key] extends readonly unknown[]
      ? T[Key]
      : TranslationShape<T[Key]>;
};

const enResources = {
  assets: enAssets,
  auth: enAuth,
  common: enCommon,
  docs: enDocs,
  editor: enEditor,
  generation: enGeneration,
  home: enHome,
  navigation: enNavigation,
  projects: enProjects,
  settings: enSettings,
  status: enStatus,
} as const;

const zhResources = {
  assets: zhAssets,
  auth: zhAuth,
  common: zhCommon,
  docs: zhDocs,
  editor: zhEditor,
  generation: zhGeneration,
  home: zhHome,
  navigation: zhNavigation,
  projects: zhProjects,
  settings: zhSettings,
  status: zhStatus,
} as const satisfies TranslationShape<typeof enResources>;

export const resources = {
  "en-US": enResources,
  "zh-CN": zhResources,
} as const satisfies Record<Language, TranslationShape<typeof enResources>>;

export function isSupportedLanguage(value: string): value is Language {
  return supportedLanguages.some((language) => language === value);
}
