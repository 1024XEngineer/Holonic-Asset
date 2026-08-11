import { beforeEach, describe, expect, it, vi } from "vitest";

import type { CreateProjectInput, ProjectSummary } from "./types";
import type { ProjectResponse } from "./project.contract";

const mocks = vi.hoisted(() => ({
  core: {
    create: vi.fn(),
    delete: vi.fn(),
    detail: vi.fn(),
    generateReference: vi.fn(),
    list: vi.fn(),
    update: vi.fn(),
  },
  deleteMockProject: vi.fn(),
  deleteMockProjectAssets: vi.fn(),
  deleteMockProjectGenerationRuns: vi.fn(),
  getMockProject: vi.fn(),
  hasMockProject: vi.fn(),
  listMockProjects: vi.fn(),
  updateMockProject: vi.fn(),
}));

vi.mock("./core-project.api", () => ({ coreProjectApi: mocks.core }));
vi.mock("./mock", () => ({
  deleteMockProject: mocks.deleteMockProject,
  getMockProject: mocks.getMockProject,
  hasMockProject: mocks.hasMockProject,
  listMockProjects: mocks.listMockProjects,
  updateMockProject: mocks.updateMockProject,
}));
vi.mock("../asset/library/mock", () => ({
  deleteMockProjectAssets: mocks.deleteMockProjectAssets,
}));
vi.mock("../generation/run/mock", () => ({
  deleteMockProjectGenerationRuns: mocks.deleteMockProjectGenerationRuns,
}));

import { projectApi, toProjectSummary } from "./project.api";

const input: CreateProjectInput = {
  name: "Moonlit Orchard",
  gameType: "Role-playing game",
  platform: "PC",
  description: "Restore the orchard.",
  reference: "reference.png",
  style: "Top-Down",
  perspective: "Top-Down",
  visualDirection: "preview.png",
};

const remoteProject: ProjectResponse = {
  id: 7,
  name: input.name,
  gameType: "Role-playing game",
  targetPlatform: "PC",
  description: input.description,
  reference: input.reference,
  style: input.style,
  perspective: input.perspective,
  userID: 4927310,
};

beforeEach(() => {
  vi.clearAllMocks();
  mocks.hasMockProject.mockReturnValue(false);
  mocks.listMockProjects.mockResolvedValue([]);
  mocks.core.create.mockResolvedValue({ id: 7 });
  mocks.core.detail.mockResolvedValue({ project: remoteProject });
  mocks.core.generateReference.mockResolvedValue({
    reference: "generated.png",
  });
  mocks.core.list.mockResolvedValue({ projects: [remoteProject] });
  mocks.core.update.mockResolvedValue({});
  mocks.core.delete.mockResolvedValue({});
});

describe("projectApi", () => {
  it("merges remote projects with mocks and falls back when listing fails", async () => {
    const mockProject = project({ id: "local" });
    mocks.listMockProjects.mockResolvedValue([mockProject]);
    mocks.core.list.mockResolvedValue({
      projects: [remoteProject, { ...remoteProject, id: "local" }],
    });

    await expect(projectApi.list()).resolves.toEqual([
      mockProject,
      toProjectSummary(remoteProject),
    ]);

    mocks.core.list.mockRejectedValueOnce(new Error("offline"));
    await expect(projectApi.list()).resolves.toEqual([mockProject]);
  });

  it("uses the correct local and remote detail, update, and deletion paths", async () => {
    const localProject = project({ id: "local" });
    mocks.hasMockProject.mockReturnValueOnce(true);
    mocks.getMockProject.mockResolvedValue(localProject);
    await expect(projectApi.detail("local")).resolves.toEqual(localProject);

    await expect(projectApi.detail("7")).resolves.toEqual(
      toProjectSummary(remoteProject),
    );

    mocks.hasMockProject.mockReturnValueOnce(true);
    mocks.updateMockProject.mockResolvedValue(localProject);
    await expect(projectApi.update(localProject)).resolves.toEqual(
      localProject,
    );

    const remoteSummary = toProjectSummary(remoteProject);
    await expect(projectApi.update(remoteSummary)).resolves.toEqual(
      remoteSummary,
    );
    expect(mocks.core.update).toHaveBeenCalledWith({
      projectID: 7,
      ...coreFields(remoteSummary),
      reference: remoteSummary.visualDirection,
    });

    mocks.hasMockProject.mockReturnValueOnce(true);
    await projectApi.delete("local");
    expect(mocks.deleteMockProjectAssets).toHaveBeenCalledWith("local");
    expect(mocks.deleteMockProjectGenerationRuns).toHaveBeenCalledWith("local");

    await projectApi.delete("7");
    expect(mocks.core.delete).toHaveBeenCalledWith({ projectID: 7 });
  });

  it("maps project inputs for creation and reference generation", async () => {
    await expect(projectApi.create(input)).resolves.toEqual({
      ...input,
      id: "7",
      assetCount: 0,
    });
    expect(mocks.core.create).toHaveBeenCalledWith({
      userID: 4927310,
      ...coreFields(input),
    });

    await expect(projectApi.generateReference(input)).resolves.toBe(
      "generated.png",
    );
    await expect(projectApi.regenerateReference(input)).resolves.toBe(
      "generated.png",
    );
    expect(mocks.core.generateReference).toHaveBeenNthCalledWith(
      1,
      coreFields(input),
    );
    expect(mocks.core.generateReference).toHaveBeenNthCalledWith(2, {
      ...coreFields(input),
      reference: "",
    });

    const customInput = { ...input, gameType: "Deck-building roguelike" };
    await projectApi.generateReference(customInput);
    expect(mocks.core.generateReference).toHaveBeenLastCalledWith(
      coreFields(customInput),
    );
  });

  it("preserves game types from the current string contract", () => {
    expect(toProjectSummary(remoteProject)).toMatchObject({
      id: "7",
      gameType: "Role-playing game",
      visualDirection: input.reference,
      assetCount: 0,
    });
    expect(
      toProjectSummary({ ...remoteProject, gameType: "Rhythm game" }),
    ).toMatchObject({ gameType: "Rhythm game" });
    expect(
      toProjectSummary({ ...remoteProject, gameType: "", targetPlatform: "" }),
    ).toMatchObject({ gameType: "", platform: "" });
  });
});

function coreFields(project: CreateProjectInput | ProjectSummary) {
  return {
    name: project.name,
    gameType: project.gameType,
    perspective: project.perspective,
    targetPlatform: "PC",
    description: project.description,
    reference: project.reference,
    style: project.style,
  };
}

function project(overrides: Partial<ProjectSummary> = {}): ProjectSummary {
  return { ...input, id: "project-1", assetCount: 0, ...overrides };
}
