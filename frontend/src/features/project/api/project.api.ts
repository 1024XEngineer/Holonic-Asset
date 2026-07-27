import {
  createMockProject,
  deleteMockProject,
  listMockProjects,
  updateMockProject,
} from "./mock";
import { deleteMockProjectAssets } from "@/features/assets/api/mock";
import { deleteMockProjectGenerationRuns } from "@/features/generation/api/mock";
import type { ProjectSummary } from "../domain";

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
