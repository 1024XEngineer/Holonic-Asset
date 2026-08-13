import { describe, expect, it } from "vitest";

import { resolveAuthRedirect } from "./auth-navigation";

describe("resolveAuthRedirect", () => {
  it("preserves internal destinations", () => {
    expect(resolveAuthRedirect("/projects/7?tab=assets#latest")).toBe(
      "/projects/7?tab=assets#latest",
    );
  });

  it.each([
    undefined,
    "",
    "https://evil.example",
    "//evil.example",
    "/\\evil.example/path",
    "/\n/evil.example/path",
    "/login",
    "/login?redirect=/projects",
  ])("falls back for unsafe or recursive destination %s", (destination) => {
    expect(resolveAuthRedirect(destination)).toBe("/projects");
  });
});
