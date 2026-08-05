import { describe, expect, it } from "vitest";

import { projectContextOptions, toCreateProjectInput } from "./project-context";

describe("toCreateProjectInput", () => {
  it("creates an API input without assigning project identity", () => {
    const input = toCreateProjectInput({
      name: "  Moonlit Orchard  ",
      gameType: "Role-playing game",
      platform: "PC",
      description: "  Restore the orchard.  ",
      perspective: "  Top-down  ",
      reference: "https://example.com/game",
      visualDirection: "data:image/png;base64,preview",
    });

    expect(input).toEqual({
      name: "Moonlit Orchard",
      gameType: "Role-playing game",
      platform: "PC",
      description: "Restore the orchard.",
      reference: "https://example.com/game",
      style: "Top-down",
      perspective: "Top-down",
      visualDirection: "data:image/png;base64,preview",
    });
    expect(input).not.toHaveProperty("id");
    expect(input).not.toHaveProperty("assetCount");
  });

  it("limits perspectives to supported game views", () => {
    expect(projectContextOptions.perspectives).toEqual([
      "Top-down",
      "Side-on",
      "Isometric",
    ]);
  });
});
