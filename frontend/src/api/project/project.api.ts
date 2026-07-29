import {
  createMockProject,
  deleteMockProject,
  getMockProject,
  listMockProjects,
  updateMockProject,
} from "./mock";
import { deleteMockProjectAssets } from "../asset/library/mock";
import { deleteMockProjectGenerationRuns } from "../generation/run/mock";
import type { ProjectSummary } from "@/model";
import { getJson, postJson } from "@/api/fetchers";

export type ProjectGameType = "RPG" | "ACT" | "SLG" | "Other";
export type ProjectViewType = "TopDown" | "SideView" | "Isometric" | "Other";
export type ProjectPlatform = "PC" | "Mobile" | "Web";

/** Matches core-api/internal/dto.ProjectResponse. */
export type ProjectResponse = {
  userID: number;
  id: number;
  name: string;
  gameType: ProjectGameType;
  viewType: ProjectViewType;
  targetPlatform: ProjectPlatform;
  description: string;
  reference: string;
  style: string;
};

export type CreateProjectRequest = Omit<ProjectResponse, "id">;
export type CreateProjectResponse = { id: number };
export type ListProjectsResponse = { projects: ProjectResponse[] };
export type ProjectDetailResponse = { project: ProjectResponse };
export type UpdateProjectRequest = {
  projectID: number;
  name?: string;
  gameType?: ProjectGameType;
  viewType?: ProjectViewType;
  targetPlatform?: ProjectPlatform;
  description?: string;
  reference?: string;
  style?: string;
};
export type UpdateProjectResponse = { success: boolean };
export type DeleteProjectRequest = { projectID: number };
export type DeleteProjectResponse = { success: boolean };

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

/** HTTP client for the routes registered by core-api. */
export const coreProjectApi = {
  create: (request: CreateProjectRequest) =>
    postJson<CreateProjectResponse>("/project/create", request),
  list: (userID: number) =>
    getJson<ListProjectsResponse>("/project/list", { userID }),
  detail: (projectID: number) =>
    getJson<ProjectDetailResponse>("/project/detail", { projectID }),
  update: (request: UpdateProjectRequest) =>
    postJson<UpdateProjectResponse>("/project/update", request),
  delete: (request: DeleteProjectRequest) =>
    postJson<DeleteProjectResponse>("/project/delete", request),
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
