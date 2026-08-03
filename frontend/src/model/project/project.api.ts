import {
  createMockProject,
  deleteMockProject,
  getMockProject,
  listMockProjects,
  updateMockProject,
} from "./mock";
import { deleteMockProjectAssets } from "../asset/library/mock";
import { deleteMockProjectGenerationRuns } from "../generation/run/mock";
import type { ProjectGameType, ProjectResponse } from "./project.contract";
import type { ProjectSummary } from "./types";

export { coreProjectApi } from "./core-project.api";
export type {
  CreateProjectRequest,
  CreateProjectResponse,
  DeleteProjectRequest,
  DeleteProjectResponse,
  ListProjectsResponse,
  ProjectDetailResponse,
  ProjectGameType,
  ProjectPlatform,
  ProjectResponse,
  ProjectViewType,
  UpdateProjectRequest,
  UpdateProjectResponse,
} from "./project.contract";

export type ProjectApi = {
  list: () => Promise<ProjectSummary[]>;
  detail: (projectId: string) => Promise<ProjectSummary>;
  create: (project: ProjectSummary) => Promise<ProjectSummary>;
  update: (project: ProjectSummary) => Promise<ProjectSummary>;
  delete: (projectId: string) => Promise<void>;
};

export const projectApi: ProjectApi = {
  list: (): Promise<ProjectSummary[]> => listMockProjects(),
  detail: (projectId: string) => getMockProject(projectId),
  create: (project: ProjectSummary) => createMockProject(project),
  update: (project: ProjectSummary) => updateMockProject(project),
  delete: async (projectId: string) => {
    await deleteMockProject(projectId);
    deleteMockProjectAssets(projectId);
    deleteMockProjectGenerationRuns(projectId);
  },
};

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
    style: project.style,
    visualStyle: project.style,
    visualDirection: "",
    assetCount,
  };
}

const projectGameTypeLabels: Record<ProjectGameType, string> = {
  RPG: "Role-playing game",
  ACT: "Action",
  SLG: "Strategy",
  Other: "Other",
};
