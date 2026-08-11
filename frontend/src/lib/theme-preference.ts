export type ThemePreference = "light" | "dark";

const themeStorageKey = "holonic-theme";

export function readThemePreference(): ThemePreference {
  return window.localStorage.getItem(themeStorageKey) === "dark"
    ? "dark"
    : "light";
}

export function applyThemePreference(theme: ThemePreference) {
  document.documentElement.classList.toggle("dark", theme === "dark");
}

export function initializeThemePreference(): ThemePreference {
  const theme = readThemePreference();
  applyThemePreference(theme);
  return theme;
}

export function saveThemePreference(theme: ThemePreference) {
  window.localStorage.setItem(themeStorageKey, theme);
  applyThemePreference(theme);
}
