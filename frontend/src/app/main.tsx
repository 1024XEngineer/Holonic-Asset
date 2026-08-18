import { StrictMode } from "react";
import { createRoot } from "react-dom/client";

import { AppProviders } from "@/app/providers";
import { router } from "@/app/router";
import { initializeAppServices } from "@/app/services";
import "@/app/styles/index.css";
import { loadAppConfig } from "@/config/app-config";
import { initializeI18n } from "@/i18n";
import { initializeThemePreference } from "@/lib/theme-preference";

await initializeI18n();
initializeThemePreference();
initializeAppServices(loadAppConfig());
createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <AppProviders router={router} />
  </StrictMode>,
);
