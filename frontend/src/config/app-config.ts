export type AppConfig = Readonly<{
  coreApi: Readonly<{
    baseUrl: string;
  }>;
}>;

export function loadAppConfig(env: ImportMetaEnv = import.meta.env): AppConfig {
  const baseUrl = env.PUBLIC_CORE_API_BASE_URL?.trim();

  return {
    coreApi: {
      baseUrl: baseUrl || "/api/v1",
    },
  };
}
