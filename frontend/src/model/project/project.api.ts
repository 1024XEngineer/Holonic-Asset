import { DataApiError } from "@/lib/data-api-error";
import { readAuthenticatedUserId } from "@/model/auth";

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
    const mockProjectIds = new Set(mockProjects.map((project) => project.id));
    try {
      const response = await coreProjectApi.list(readAuthenticatedUserId());
      const remoteProjects = response.projects
        .map((project) => toProjectSummary(project))
        .filter((project) => !mockProjectIds.has(project.id));
      return [...mockProjects, ...remoteProjects];
    } catch (error) {
      if (error instanceof DataApiError && error.code === "UNAVAILABLE") {
        return mockProjects;
      }
      throw error;
    }
  },
  detail: async (projectId) => {
    if (hasMockProject(projectId)) return getMockProject(projectId);

    const response = await coreProjectApi.detail(Number(projectId));
    return toProjectSummary(response.project);
  },
  create: async (input) => {
    const response = await coreProjectApi.create({
      userID: readAuthenticatedUserId(),
      ...toCoreProjectFields(input),
    });

    return {
      ...input,
      id: String(response.id),
    };
  },
  generateReference: async (input) => {
    const response = await coreProjectApi.generateReference(
      toCoreProjectFields(input),
    );
    return response.reference;
  },
  regenerateReference: async (input) => {
    const response = await coreProjectApi.generateReference(
      toCoreProjectFields(input),
    );
    return response.reference;
  },
  update: async (project) => {
    if (hasMockProject(project.id)) return updateMockProject(project);

    await coreProjectApi.update({
      projectID: Number(project.id),
      ...toCoreProjectFields(project),
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

function toCoreProjectFields(
  input: CreateProjectInput,
): GenerateProjectReferenceRequest {
  return {
    name: input.name,
    gameType: input.gameType,
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
  assetCount?: number,
): ProjectSummary {
  const summary: ProjectSummary = {
    id: String(project.id),
    name: project.name,
    gameType: project.gameType,
    platform: project.targetPlatform,
    description: project.description,
    reference: project.reference,
    style: project.style,
    perspective: project.perspective,
  };
  return assetCount === undefined ? summary : { ...summary, assetCount };
}
