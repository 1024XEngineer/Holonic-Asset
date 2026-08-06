import { perspectiveOptions } from "@/model/project";
import type { CreateProjectInput, ProjectSummary } from "@/model/project";

import type {
  NewProjectDraft,
  ProjectSettingsDraft,
} from "../types/project-draft";

export const projectContextOptions = {
  gameTypes: [
    "Role-playing game",
    "Platformer",
    "Puzzle",
    "Strategy",
    "Simulation",
  ],
  perspectives: perspectiveOptions,
  platforms: ["PC", "Mobile", "Web", "Console", "Multi-platform"],
} as const;

export const editableProjectContextOptions = {
  gameTypes: [...projectContextOptions.gameTypes, "Other"],
} as const;

export function createNewProjectDraft(): NewProjectDraft {
  return {
    name: "",
    gameType: projectContextOptions.gameTypes[0],
    platform: projectContextOptions.platforms[0],
    description: "",
    perspective: projectContextOptions.perspectives[0],
    reference: "",
  };
}

export function toCreateProjectInput(
  draft: NewProjectDraft & { visualDirection?: string },
): CreateProjectInput {
  return {
    name: draft.name.trim(),
    gameType: draft.gameType,
    platform: draft.platform,
    description: draft.description.trim() || "A new game asset workspace.",
    reference: draft.reference.trim(),
    style: draft.perspective,
    perspective: draft.perspective,
    visualDirection: draft.visualDirection ?? "",
  };
}

export function createProjectSettingsDraft(
  project: ProjectSummary,
): ProjectSettingsDraft {
  const hasKnownGameType = isKnownOption(
    projectContextOptions.gameTypes,
    project.gameType,
  );
  return {
    name: project.name,
    gameType: hasKnownGameType ? project.gameType : "Other",
    customGameType: hasKnownGameType ? "" : project.gameType,
    perspective: project.perspective,
    platform: project.platform,
    description: project.description,
    visualDirection: project.visualDirection,
  };
}

export function applyProjectSettings(
  project: ProjectSummary,
  draft: ProjectSettingsDraft,
): ProjectSummary | undefined {
  const name = draft.name.trim();
  const gameType = resolveEditableOption(draft.gameType, draft.customGameType);
  if (!name || !gameType) return undefined;

  return {
    ...project,
    name,
    gameType,
    perspective: draft.perspective,
    style: draft.perspective,
    platform: draft.platform,
    description: draft.description,
    visualDirection: draft.visualDirection,
  };
}

function isKnownOption(options: readonly string[], value: string) {
  return options.includes(value);
}

function resolveEditableOption(value: string, customValue: string) {
  return value === "Other" ? customValue.trim() : value;
}
