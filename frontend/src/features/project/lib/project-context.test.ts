import { describe, expect, it } from "vitest";

import { toCreateProjectInput } from "./project-context";

describe("toCreateProjectInput", () => {
  it("creates an API input without assigning project identity", () => {
    const input = toCreateProjectInput({
      name: "  Moonlit Orchard  ",
      gameType: "Role-playing game",
      platform: "PC",
      description: "  Restore the orchard.  ",
      visualStyle: "  Pixel art  ",
      visualDirection: "data:image/png;base64,preview",
    });

    expect(input).toEqual({
      name: "Moonlit Orchard",
      gameType: "Role-playing game",
      platform: "PC",
      description: "Restore the orchard.",
      style: "Pixel art",
      visualStyle: "Pixel art",
      visualDirection: "data:image/png;base64,preview",
    });
    expect(input).not.toHaveProperty("id");
    expect(input).not.toHaveProperty("assetCount");
  });
});
