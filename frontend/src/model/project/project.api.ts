import {
  deleteMockProject,
  getMockProject,
  hasMockProject,
  listMockProjects,
  updateMockProject,
} from "./mock";
import { deleteMockProjectAssets } from "../asset/library/mock";
import { deleteMockProjectGenerationRuns } from "../generation/run/mock";
import { coreProjectApi } from "./core-project.api";
import type {
  GenerateProjectReferenceRequest,
  ProjectGameType,
  ProjectResponse,
} from "./project.contract";
import type { CreateProjectInput, ProjectSummary } from "./types";

export { coreProjectApi } from "./core-project.api";
export type {
  CreateProjectRequest,
  CreateProjectResponse,
  DeleteProjectRequest,
  DeleteProjectResponse,
  GenerateProjectReferenceRequest,
  GenerateProjectReferenceResponse,
  ListProjectsResponse,
  ProjectDetailResponse,
  ProjectGameType,
  ProjectPlatform,
  ProjectPerspective,
  ProjectResponse,
  UpdateProjectRequest,
  UpdateProjectResponse,
} from "./project.contract";

export type ProjectApi = {
  list: () => Promise<ProjectSummary[]>;
  detail: (projectId: string) => Promise<ProjectSummary>;
  create: (input: CreateProjectInput) => Promise<ProjectSummary>;
  generateReference: (input: CreateProjectInput) => Promise<string>;
  regenerateReference: (input: CreateProjectInput) => Promise<string>;
  update: (project: ProjectSummary) => Promise<ProjectSummary>;
  delete: (projectId: string) => Promise<void>;
};

export const projectApi: ProjectApi = {
  list: async () => {
    const mockProjects = await listMockProjects();
    try {
      const response = await coreProjectApi.list(coreApiUserId);
      const remoteProjects = response.projects
        .map((project) => toProjectSummary(project))
        .filter(
          (project) => !mockProjects.some((mock) => mock.id === project.id),
        );
      return [...mockProjects, ...remoteProjects];
    } catch {
      return mockProjects;
    }
  },
  detail: async (projectId) => {
    if (hasMockProject(projectId)) return getMockProject(projectId);

    const response = await coreProjectApi.detail(Number(projectId));
    return toProjectSummary(response.project);
  },
  create: async (input: CreateProjectInput) => {
    const response = await coreProjectApi.create({
      userID: coreApiUserId,
      ...toCoreProjectFields(input),
    });

    return {
      ...input,
      id: String(response.id),
      assetCount: 0,
    };
  },
  generateReference: async (input) => {
    const response = await coreProjectApi.generateReference(
      toCoreProjectFields(input),
    );
    return response.reference;
  },
  regenerateReference: async (input) => {
    const response = await coreProjectApi.generateReference({
      ...toCoreProjectFields(input),
      reference: "",
    });
    return response.reference;
  },
  update: async (project) => {
    if (hasMockProject(project.id)) return updateMockProject(project);

    await coreProjectApi.update({
      projectID: Number(project.id),
      ...toCoreProjectFields(project),
      reference: project.visualDirection || project.reference,
    });
    return project;
  },
  delete: async (projectId) => {
    if (hasMockProject(projectId)) {
      await deleteMockProject(projectId);
      deleteMockProjectAssets(projectId);
      deleteMockProjectGenerationRuns(projectId);
      return;
    }

    await coreProjectApi.delete({ projectID: Number(projectId) });
  },
};

const gameTypeByLabel: Record<string, ProjectGameType> = {
  "Role-playing game": "RPG",
  Action: "ACT",
  Platformer: "ACT",
  Strategy: "SLG",
};

const coreApiUserId = Number(
  import.meta.env.PUBLIC_CORE_API_USER_ID ?? "4927310",
);

function toCoreGameType(value: string): ProjectGameType {
  return gameTypeByLabel[value] ?? "";
}

function toCoreProjectFields(
  input: CreateProjectInput,
): GenerateProjectReferenceRequest {
  return {
    name: input.name,
    gameType: toCoreGameType(input.gameType),
    perspective: input.perspective,
    targetPlatform: toCorePlatform(input.platform),
    description: input.description,
    reference: input.reference,
    style: input.style,
  };
}

function toCorePlatform(value: string): ProjectResponse["targetPlatform"] {
  return value === "PC" || value === "Mobile" || value === "Web" ? value : "";
}

export function toProjectSummary(
  project: ProjectResponse,
  assetCount = 0,
): ProjectSummary {
  return {
    id: String(project.id),
    name: project.name,
    gameType: projectGameTypeLabels[project.gameType],
    platform: project.targetPlatform,
    description: project.description,
    reference: project.reference,
    style: project.style,
    perspective: project.perspective,
    visualDirection: project.reference,
    assetCount,
  };
}

const projectGameTypeLabels: Record<ProjectGameType, string> = {
  RPG: "Role-playing game",
  ACT: "Action",
  SLG: "Strategy",
  "": "Unspecified",
};
