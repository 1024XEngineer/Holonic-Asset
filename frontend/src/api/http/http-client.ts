import { DataApiError } from "@/lib/data-api-error";

export type HttpClientConfig = {
  baseUrl: string;
};

export type HttpRequest = {
  path: string;
  method: "GET" | "POST";
  /** GET parameters; POST requests send this value as JSON. */
  data: Record<string, unknown>;
};

export type HttpRequestExecutor = <TResponse>(
  request: HttpRequest,
) => Promise<TResponse>;

/** Creates a JSON HTTP executor shared by API domain modules. */
export function createHttpExecutor(
  config: HttpClientConfig,
): HttpRequestExecutor {
  return async <TResponse>({ path, method, data }: HttpRequest) => {
    const url = new URL(path, `${config.baseUrl.replace(/\/$/, "")}/`);
    const init: RequestInit = { method, headers: { Accept: "application/json" } };

    if (method === "GET") {
      for (const [key, value] of Object.entries(data)) {
        url.searchParams.set(key, String(value));
      }
    } else {
      init.headers = { ...init.headers, "Content-Type": "application/json" };
      init.body = JSON.stringify(data);
    }

    let response: Response;
    try {
      response = await fetch(url, init);
    } catch (error) {
      throw new DataApiError("UNAVAILABLE", "Unable to reach the API.", error);
    }

    const body: unknown = await response.json().catch(() => undefined);
    if (!response.ok) {
      throw new DataApiError(
        response.status === 404 ? "NOT_FOUND" : "UNKNOWN",
        `API request failed (${response.status}).`,
        body,
      );
    }
    return body as TResponse;
  };
}
