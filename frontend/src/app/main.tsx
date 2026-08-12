import { StrictMode } from "react";
import { createRoot } from "react-dom/client";

import { AppProviders } from "@/app/providers";
import { router } from "@/app/router";
import "@/app/styles/index.css";
import { clearAuthSession, readAccessToken } from "@/features/auth";
import { initializeI18n } from "@/i18n";
import { initializeThemePreference } from "@/lib/theme-preference";
import { configureCoreApiAuth } from "@/model/fetchers";

await initializeI18n();
initializeThemePreference();
configureCoreApiAuth({
  getAccessToken: readAccessToken,
  onUnauthorized: clearAuthSession,
});
createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <AppProviders router={router} />
  </StrictMode>,
);
