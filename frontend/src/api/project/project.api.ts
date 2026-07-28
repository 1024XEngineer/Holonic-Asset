import {
  createMockProject,
  deleteMockProject,
  listMockProjects,
  updateMockProject,
} from "./mock";
import { deleteMockProjectAssets } from "../asset/library/mock";
import { deleteMockProjectGenerationRuns } from "../generation/run/mock";
import type { ProjectSummary } from "@/model";

export const projectApi = {
  list: (): Promise<ProjectSummary[]> => listMockProjects(),
  create: (project: ProjectSummary) => createMockProject(project),
  update: (project: ProjectSummary) => updateMockProject(project),
  delete: async (projectId: string) => {
    await deleteMockProject(projectId);
    deleteMockProjectAssets(projectId);
    deleteMockProjectGenerationRuns(projectId);
  },
};
