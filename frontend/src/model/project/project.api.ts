import {
  createMockProject,
  deleteMockProject,
  getMockProject,
  listMockProjects,
  updateMockProject,
} from "./mock";
import { deleteMockProjectAssets } from "../asset/library/mock";
import { deleteMockProjectGenerationRuns } from "../generation/run/mock";
import type {
  ProjectGameType,
  ProjectPerspective,
  ProjectResponse,
} from "./project.contract";
import type { CreateProjectInput, ProjectSummary } from "./types";

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
  ProjectPerspective,
  ProjectResponse,
  UpdateProjectRequest,
  UpdateProjectResponse,
} from "./project.contract";

export type ProjectApi = {
  list: () => Promise<ProjectSummary[]>;
  detail: (projectId: string) => Promise<ProjectSummary>;
  create: (input: CreateProjectInput) => Promise<ProjectSummary>;
  update: (project: ProjectSummary) => Promise<ProjectSummary>;
  delete: (projectId: string) => Promise<void>;
};

export const projectApi: ProjectApi = {
  list: (): Promise<ProjectSummary[]> => listMockProjects(),
  detail: (projectId: string) => getMockProject(projectId),
  create: (input: CreateProjectInput) => createMockProject(input),
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
    reference: project.reference,
    style: project.style,
    perspective: projectPerspectiveLabels[project.perspective],
    visualDirection: "",
    assetCount,
  };
}

const projectGameTypeLabels: Record<ProjectGameType, string> = {
  RPG: "Role-playing game",
  ACT: "Action",
  SLG: "Strategy",
  "": "Unspecified",
};

const projectPerspectiveLabels: Record<ProjectPerspective, string> = {
  TopDown: "Top-down",
  SideOn: "Side-On",
  Isometric: "Isometric",
  "": "Top-down",
};
