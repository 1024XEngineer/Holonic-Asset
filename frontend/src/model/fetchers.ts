import { DataApiError } from "@/lib/data-api-error";

type QueryPrimitive = string | number | boolean;
type QueryValue = QueryPrimitive | readonly QueryPrimitive[] | null | undefined;

export type ApiResponse<T> = {
  code: number;
  message: string;
  data: T;
};

const apiBaseUrl = import.meta.env.PUBLIC_CORE_API_BASE_URL ?? "/api/v1";

function getJson<T>(
  path: string,
  query?: Record<string, QueryValue>,
): Promise<T> {
  return requestJson<T>(withQuery(path, query));
}

function postJson<T>(path: string, body?: unknown): Promise<T> {
  return requestJson<T>(path, { method: "POST" }, body);
}

export async function getEnvelope<T>(
  path: string,
  query?: Record<string, QueryValue>,
): Promise<T> {
  return unwrapEnvelope(await getJson<ApiResponse<T>>(path, query));
}

export async function postEnvelope<T>(
  path: string,
  body?: unknown,
): Promise<T> {
  return unwrapEnvelope(await postJson<ApiResponse<T>>(path, body));
}

export async function putEnvelope<T>(path: string, body?: unknown): Promise<T> {
  return unwrapEnvelope(
    await requestJson<ApiResponse<T>>(path, { method: "PUT" }, body),
  );
}

export async function deleteEnvelope<T>(
  path: string,
  body?: unknown,
): Promise<T> {
  return unwrapEnvelope(
    await requestJson<ApiResponse<T>>(path, { method: "DELETE" }, body),
  );
}

function asyncRequestInit(body?: unknown): RequestInit {
  return body === undefined
    ? {}
    : {
        body: JSON.stringify(body),
        headers: { "Content-Type": "application/json" },
      };
}

async function requestJson<T>(
  path: string,
  init: RequestInit = {},
  body?: unknown,
): Promise<T> {
  let response: Response;
  try {
    response = await fetch(`${apiBaseUrl}${path}`, {
      ...asyncRequestInit(body),
      ...init,
    });
  } catch (error) {
    throw new DataApiError("UNAVAILABLE", "Unable to reach the API.", error);
  }

  const responseBody = await parseResponse(response);

  if (!response.ok) {
    throw new DataApiError(
      dataApiErrorCodeForStatus(response.status),
      `API request failed (${response.status}).`,
      responseBody,
    );
  }

  return responseBody as T;
}

async function parseResponse(response: Response): Promise<unknown> {
  if (response.status === 204) return undefined;

  const text = await response.text();
  if (!text) return undefined;

  try {
    return JSON.parse(text) as unknown;
  } catch {
    return text;
  }
}

function unwrapEnvelope<T>(response: ApiResponse<T>): T {
  if (response.code !== 200) {
    throw new DataApiError(
      dataApiErrorCodeForStatus(response.code),
      response.message || "Request failed",
      response,
    );
  }
  return response.data;
}

function dataApiErrorCodeForStatus(status: number) {
  if (status === 400 || status === 422) return "BAD_REQUEST" as const;
  if (status === 404) return "NOT_FOUND" as const;
  if (status === 409) return "CONFLICT" as const;
  return "UNKNOWN" as const;
}

function withQuery(
  path: string,
  query: Record<string, QueryValue> | undefined,
) {
  if (!query) return path;

  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(query)) {
    if (Array.isArray(value)) {
      for (const item of value) params.append(key, String(item));
      continue;
    }
    if (value !== undefined && value !== null) params.set(key, String(value));
  }
  const search = params.toString();
  return search ? `${path}?${search}` : path;
}
