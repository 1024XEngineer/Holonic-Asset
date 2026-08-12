import createClient, { type Middleware } from "openapi-fetch";

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

let coreApiAuth: CoreApiAuth = {
  getAccessToken: () => undefined,
  onUnauthorized: () => undefined,
};

export function configureCoreApiAuth(auth: CoreApiAuth) {
  coreApiAuth = auth;
}

export const coreApiClient = createClient<paths>({
  baseUrl: apiBaseUrl(),
  fetch: (request) => fetch(request),
});

const authMiddleware: Middleware = {
  onRequest({ request, schemaPath }) {
    if (schemaPath === "/auth/login") return;

    const accessToken = coreApiAuth.getAccessToken();
    if (accessToken && !request.headers.has("Authorization")) {
      request.headers.set("Authorization", `Bearer ${accessToken}`);
    }
  },
  onResponse({ response }) {
    if (response.status === 401) coreApiAuth.onUnauthorized();
  },
  onError({ error }) {
    return new DataApiError("UNAVAILABLE", "Unable to reach the API.", error);
  },
};

coreApiClient.use(authMiddleware);

export function unwrapApiResponse<T>({ data, error, response }: ApiResult): T {
  if (!response.ok) {
    throw new DataApiError(
      dataApiErrorCodeForStatus(response.status),
      `API request failed (${response.status}).`,
      error,
    );
  }

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

function apiBaseUrl() {
  const configuredUrl = import.meta.env.PUBLIC_CORE_API_BASE_URL ?? "/api/v1";
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
