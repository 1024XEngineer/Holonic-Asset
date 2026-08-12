import { describe, expect, it } from "vitest";

import { resolveAuthRedirect } from "./auth-navigation";

describe("resolveAuthRedirect", () => {
  it("preserves internal destinations", () => {
    expect(resolveAuthRedirect("/projects/7?tab=assets")).toBe(
      "/projects/7?tab=assets",
    );
  });

  it.each([undefined, "", "https://evil.example", "//evil.example", "/login"])(
    "falls back for unsafe or recursive destination %s",
    (destination) => {
      expect(resolveAuthRedirect(destination)).toBe("/projects");
    },
  );
});
