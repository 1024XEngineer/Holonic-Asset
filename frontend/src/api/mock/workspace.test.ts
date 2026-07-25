import { afterEach, describe, expect, it } from "vitest";

import {
  createMockGenerationRun,
  deleteMockProject,
  listMockAssetGroups,
  listMockGenerationRuns,
  listMockProjects,
  resetMockWorkspace,
  saveMockAssetRevision,
} from "./workspace";

afterEach(resetMockWorkspace);

describe("Mock Workspace", () => {
  it("removes all project-scoped state when a project is deleted", async () => {
    const projectId = "moonlit-orchard";
    createMockGenerationRun({
      projectId,
      kind: "character",
      name: "New Hero",
      prompt: "A tiny adventurer",
      canvasSize: "32 × 32 px",
      useProjectContext: true,
    });

    await deleteMockProject(projectId);

    expect(await listMockProjects()).not.toContainEqual(
      expect.objectContaining({ id: projectId }),
    );
    expect(await listMockAssetGroups(projectId)).toEqual([]);
    expect(await listMockGenerationRuns(projectId)).toEqual([]);
  });

  it("resets to cloned in-memory seed state", async () => {
    const projects = await listMockProjects();
    projects[0]!.name = "Changed outside the workspace";

    expect((await listMockProjects())[0]!.name).toBe("Moonlit Orchard");

    await deleteMockProject("moonlit-orchard");
    resetMockWorkspace();

    expect(await listMockAssetGroups("moonlit-orchard")).not.toEqual([]);
    expect(await listMockGenerationRuns("moonlit-orchard")).toEqual([]);
  });

  it("rejects a record document whose mode does not match the asset kind", async () => {
    await expect(
      saveMockAssetRevision("moonlit-orchard", "forager-hero", {
        mode: "sprite-sheet",
        prompt: "Invalid mode",
        spriteSheet: { gridSize: 8, items: [] },
      }),
    ).rejects.toThrow("Record content does not match the asset kind.");
  });
});
