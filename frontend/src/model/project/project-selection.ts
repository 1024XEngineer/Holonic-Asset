import type { ProjectSummary } from "./types";

const LAST_PROJECT_STORAGE_KEY = "game-asset-pack:last-project-id";

export function reconcileProjectSelection(
  projects: ProjectSummary[],
  requestedProjectId: string | undefined,
): { redirectProjectId?: string } {
  const requestedProject = projects.find(
    (project) => project.id === requestedProjectId,
  );
  if (requestedProject) {
    writeLastProjectId(requestedProject.id);
    return {};
  }

  if (requestedProjectId) {
    clearLastProjectId(requestedProjectId);
    return {};
  }

  const rememberedProjectId = readLastProjectId();
  if (!rememberedProjectId) return {};

  if (projects.some((project) => project.id === rememberedProjectId)) {
    return { redirectProjectId: rememberedProjectId };
  }

  clearLastProjectId(rememberedProjectId);
  return {};
}

export function removeProjectSelection(
  projects: ProjectSummary[],
  removedProjectId: string,
  selectedProjectId: string | undefined,
) {
  clearLastProjectId(removedProjectId);
  return selectedProjectId === removedProjectId
    ? projects.find((project) => project.id !== removedProjectId)?.id
    : undefined;
}

export function readLastProjectId() {
  try {
    return localStorage.getItem(LAST_PROJECT_STORAGE_KEY) ?? undefined;
  } catch {
    return undefined;
  }
}

function writeLastProjectId(projectId: string) {
  try {
    localStorage.setItem(LAST_PROJECT_STORAGE_KEY, projectId);
  } catch {
    // Project selection still works when browser storage is unavailable.
  }
}

function clearLastProjectId(projectId?: string) {
  try {
    if (
      projectId === undefined ||
      localStorage.getItem(LAST_PROJECT_STORAGE_KEY) === projectId
    ) {
      localStorage.removeItem(LAST_PROJECT_STORAGE_KEY);
    }
  } catch {
    // Project selection still works when browser storage is unavailable.
  }
}
