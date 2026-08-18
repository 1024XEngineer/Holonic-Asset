import { describe, expect, it } from "vitest";

import { loadAppConfig } from "./app-config";

describe("loadAppConfig", () => {
  it("maps the public API URL into the application config", () => {
    expect(
      loadAppConfig({
        PUBLIC_CORE_API_BASE_URL: " https://api.example.test/v1 ",
      }),
    ).toEqual({
      coreApi: { baseUrl: "https://api.example.test/v1" },
    });
  });

  it("uses the same-origin API path when the URL is absent", () => {
    expect(loadAppConfig({})).toEqual({
      coreApi: { baseUrl: "/api/v1" },
    });
  });
});
