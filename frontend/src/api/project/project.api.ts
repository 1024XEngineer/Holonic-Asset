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
import type { ProjectApi } from "./project";

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
