import { clearAuthSession, readAccessToken } from "@/model/auth";
import { configureCoreApi, configureCoreApiAuth } from "@/model/fetchers";
import type { AppConfig } from "@/config/app-config";

export function initializeAppServices(config: AppConfig) {
  configureCoreApi(config.coreApi);
  configureCoreApiAuth({
    getAccessToken: readAccessToken,
    onUnauthorized: clearAuthSession,
  });
}
