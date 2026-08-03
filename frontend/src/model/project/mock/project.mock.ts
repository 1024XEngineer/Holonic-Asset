import type { ProjectSummary } from "../types";
import { projectSummaries } from "./project.seed";

let projects = createProjectState();

function createProjectState() {
  return structuredClone(projectSummaries);
}

export async function listMockProjects() {
  return structuredClone(projects);
}

export async function getMockProject(projectId: string) {
  const project = projects.find((item) => item.id === projectId);
  if (!project) throw new Error(`Project not found: ${projectId}`);
  return structuredClone(project);
}

export async function createMockProject(project: ProjectSummary) {
  projects = [...projects, structuredClone(project)];
  return structuredClone(project);
}

export async function updateMockProject(project: ProjectSummary) {
  projects = projects.map((item) =>
    item.id === project.id ? structuredClone(project) : item,
  );
  return structuredClone(project);
}

export async function deleteMockProject(projectId: string) {
  projects = projects.filter((project) => project.id !== projectId);
}

export function hasMockProject(projectId: string) {
  return projects.some((project) => project.id === projectId);
}

export function resetMockProjects() {
  projects = createProjectState();
}
