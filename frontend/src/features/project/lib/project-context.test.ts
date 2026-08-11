import { describe, expect, it } from "vitest";

import type { ProjectSummary } from "@/model/project";

import {
  applyProjectSettings,
  createNewProjectDraft,
  createProjectSettingsDraft,
  projectContextOptions,
  toCreateProjectInput,
} from "./project-context";

describe("toCreateProjectInput", () => {
  it("creates an API input without assigning project identity", () => {
    const input = toCreateProjectInput({
      name: "  Moonlit Orchard  ",
      gameType: "Role-playing game",
      platform: "PC",
      description: "  Restore the orchard.  ",
      perspective: "Top-Down",
      reference: "https://example.com/game",
      visualDirection: "data:image/png;base64,preview",
    });

    expect(input).toEqual({
      name: "Moonlit Orchard",
      gameType: "Role-playing game",
      platform: "PC",
      description: "Restore the orchard.",
      reference: "https://example.com/game",
      style: "Top-Down",
      perspective: "Top-Down",
      visualDirection: "data:image/png;base64,preview",
    });
    expect(input).not.toHaveProperty("id");
    expect(input).not.toHaveProperty("assetCount");
  });

  it("limits perspectives to supported game views", () => {
    expect(projectContextOptions.perspectives).toEqual([
      "Top-Down",
      "Side-On",
      "Isometric",
    ]);
  });

  it("rejects unsupported project options", () => {
    expect(() =>
      toCreateProjectInput({
        name: "Test",
        gameType: "Unknown",
        platform: "PC",
        description: "",
        perspective: "Top-Down",
        reference: "",
      } as never),
    ).toThrow();
  });

  it("creates a complete draft and fills optional API defaults", () => {
    const draft = createNewProjectDraft();

    expect(draft).toEqual({
      name: "",
      gameType: "Role-playing game",
      platform: "PC",
      description: "",
      perspective: "Top-Down",
      reference: "",
    });
    expect(
      toCreateProjectInput({ ...draft, name: "New project" }),
    ).toMatchObject({
      description: "A new game asset workspace.",
      visualDirection: "",
    });
  });

  it("maps known and custom game types into editable drafts", () => {
    expect(createProjectSettingsDraft(project())).toMatchObject({
      gameType: "Role-playing game",
      customGameType: "",
    });
    expect(
      createProjectSettingsDraft(project({ gameType: "Rhythm game" })),
    ).toMatchObject({ gameType: "Other", customGameType: "Rhythm game" });
  });

  it("applies valid settings and rejects blank identity fields", () => {
    const current = project();
    const knownDraft = createProjectSettingsDraft(current);
    knownDraft.name = "  Updated project  ";
    knownDraft.perspective = "Isometric";

    expect(applyProjectSettings(current, knownDraft)).toMatchObject({
      name: "Updated project",
      gameType: "Role-playing game",
      perspective: "Isometric",
      style: "Isometric",
    });

    const customDraft = {
      ...knownDraft,
      gameType: "Other",
      customGameType: "  Roguelike  ",
    };
    expect(applyProjectSettings(current, customDraft)?.gameType).toBe(
      "Roguelike",
    );
    expect(
      applyProjectSettings(current, { ...customDraft, name: " " }),
    ).toBeUndefined();
    expect(
      applyProjectSettings(current, { ...customDraft, customGameType: " " }),
    ).toBeUndefined();
  });
});

function project(overrides: Partial<ProjectSummary> = {}): ProjectSummary {
  return {
    id: "project-1",
    name: "Moonlit Orchard",
    style: "Top-Down",
    gameType: "Role-playing game",
    platform: "PC",
    description: "Restore the orchard.",
    reference: "",
    perspective: "Top-Down",
    visualDirection: "",
    assetCount: 0,
    ...overrides,
  };
}
