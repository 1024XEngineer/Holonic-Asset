import { describe, expect, it } from "vitest";

import { isSafeImageUrl } from "./is-safe-image-url";

describe("isSafeImageUrl", () => {
  it.each(["/assets/sky.png", "https://cdn.example.com/sky.png"])(
    "accepts %s",
    (value) => {
      expect(isSafeImageUrl(value)).toBe(true);
    },
  );

  it.each([
    "",
    "//cdn.example.com/sky.png",
    "http://cdn.example.com/sky.png",
    "javascript:alert(1)",
    "data:image/png;base64,abc",
  ])("rejects %s", (value) => {
    expect(isSafeImageUrl(value)).toBe(false);
  });

  it.each([undefined, null, 42, {}])("rejects non-string value %p", (value) => {
    expect(isSafeImageUrl(value)).toBe(false);
  });
});
