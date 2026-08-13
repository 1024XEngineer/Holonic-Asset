import { DataApiError } from "@/lib/data-api-error";
import { readAuthenticatedUserId } from "@/model/auth";

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
    try {
      const response = await coreProjectApi.list(readAuthenticatedUserId());
      return response.projects.map((project) => toProjectSummary(project));
    } catch (error) {
      if (error instanceof DataApiError && error.code === "UNAVAILABLE") {
        return [];
      }
      throw error;
    }
  },
  detail: async (projectId) => {
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
    await coreProjectApi.update({
      projectID: Number(project.id),
      ...toCoreProjectFields(project),
    });
    return project;
  },
  delete: async (projectId) => {
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
