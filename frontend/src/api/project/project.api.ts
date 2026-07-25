import {
  createMockProject,
  deleteMockProject,
  listMockProjects,
  updateMockProject,
} from "@/api/mock";
import type { ProjectSummary } from "@/types/project";

export const projectApi = {
  list: (): Promise<ProjectSummary[]> => listMockProjects(),
  create: (project: ProjectSummary) => createMockProject(project),
  update: (project: ProjectSummary) => updateMockProject(project),
  delete: (projectId: string) => deleteMockProject(projectId),
};
