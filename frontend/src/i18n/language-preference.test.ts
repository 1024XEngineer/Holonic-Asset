import { describe, expect, it } from "vitest";

import { resolveLanguagePreference } from "./language-preference";

describe("resolveLanguagePreference", () => {
  it("uses a supported stored language before browser preferences", () => {
    expect(resolveLanguagePreference("en-US", ["zh-CN"])).toBe("en-US");
  });

  it("matches supported languages from the browser preference list", () => {
    expect(resolveLanguagePreference(null, ["fr-FR", "zh-Hant-TW"])).toBe(
      "zh-CN",
    );
    expect(resolveLanguagePreference("invalid", ["en-GB", "zh-CN"])).toBe(
      "en-US",
    );
  });

  it("falls back to the default language", () => {
    expect(resolveLanguagePreference(null, ["fr-FR"])).toBe("en-US");
  });
});
