import createClient, { type Client, type Middleware } from "openapi-fetch";

import { DataApiError } from "@/lib/data-api-error";
import type { paths } from "@/model/generated/core-api";

type ApiResponse = {
  code: number;
  message: string;
  data: unknown;
};

type ApiResult = {
  data?: unknown;
  error?: unknown;
  response: Response;
};

type CoreApiAuth = {
  getAccessToken: () => string | undefined;
  onUnauthorized: () => void;
};

export type CoreApiConfig = {
  readonly baseUrl: string;
  readonly fetch?: typeof fetch;
};

export type CoreApiClient = Client<paths>;

export type CoreApiClients = {
  public: CoreApiClient;
  authenticated: CoreApiClient;
};

let coreApiAuth: CoreApiAuth = {
  getAccessToken: () => undefined,
  onUnauthorized: () => undefined,
};

export function configureCoreApiAuth(auth: CoreApiAuth) {
  coreApiAuth = auth;
}

const unavailableMiddleware: Middleware = {
  onError({ error }) {
    return new DataApiError("UNAVAILABLE", "Unable to reach the API.", error);
  },
};

const authMiddleware: Middleware = {
  onRequest({ request }) {
    const accessToken = coreApiAuth.getAccessToken();
    if (accessToken && !request.headers.has("Authorization")) {
      request.headers.set("Authorization", `Bearer ${accessToken}`);
    }
  },
  onResponse({ response }) {
    if (response.status === 401) coreApiAuth.onUnauthorized();
  },
};

const defaultApiClients = createCoreApiClients({ baseUrl: "/api/v1" });

export let publicCoreApiClient = defaultApiClients.public;
export let coreApiClient = defaultApiClients.authenticated;

export function createCoreApiClients(config: CoreApiConfig): CoreApiClients {
  return {
    public: createCoreApiClient(config),
    authenticated: createCoreApiClient(config, authMiddleware),
  };
}

export function configureCoreApi(config: CoreApiConfig) {
  const clients = createCoreApiClients(config);
  publicCoreApiClient = clients.public;
  coreApiClient = clients.authenticated;
}

function createCoreApiClient(config: CoreApiConfig, middleware?: Middleware) {
  const client = createClient<paths>({
    baseUrl: resolveApiBaseUrl(config.baseUrl),
    fetch: (request) => (config.fetch ?? fetch)(request),
  });
  client.use(unavailableMiddleware);
  if (middleware) client.use(middleware);
  return client;
}

export function ensureApiResponseSuccess({ error, response }: ApiResult) {
  if (!response.ok) {
    throw new DataApiError(
      dataApiErrorCodeForStatus(response.status),
      `API request failed (${response.status}).`,
      error,
    );
  }
}

export function unwrapApiResponse<T>(result: ApiResult): T {
  ensureApiResponseSuccess(result);
  const { data } = result;

  if (!isApiResponse(data)) {
    throw new DataApiError("UNKNOWN", "Invalid API response.", data);
  }
  if (data.code !== 200) {
    throw new DataApiError(
      dataApiErrorCodeForStatus(data.code),
      data.message || "Request failed",
      data,
    );
  }
  return data.data as T;
}

function resolveApiBaseUrl(configuredUrl: string) {
  if (/^https?:\/\//i.test(configuredUrl)) return configuredUrl;

  const origin =
    typeof location === "undefined" ? "http://localhost" : location.origin;
  return new URL(configuredUrl, origin).toString();
}

function isApiResponse(value: unknown): value is ApiResponse {
  if (!value || typeof value !== "object") return false;
  const response = value as Partial<ApiResponse>;
  return (
    typeof response.code === "number" &&
    typeof response.message === "string" &&
    "data" in response
  );
}

function dataApiErrorCodeForStatus(status: number) {
  if (status === 400 || status === 422) return "BAD_REQUEST" as const;
  if (status === 401) return "UNAUTHORIZED" as const;
  if (status === 404) return "NOT_FOUND" as const;
  if (status === 409) return "CONFLICT" as const;
  return "UNKNOWN" as const;
}
