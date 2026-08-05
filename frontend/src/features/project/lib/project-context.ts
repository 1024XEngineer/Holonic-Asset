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
  perspectives: ["Top-down", "Side-on", "Isometric"],
  platforms: ["PC", "Mobile", "Web", "Console", "Multi-platform"],
} as const;

export const editableProjectContextOptions = {
  gameTypes: [...projectContextOptions.gameTypes, "Other"],
  perspectives: [...projectContextOptions.perspectives, "Other"],
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
  const perspective = draft.perspective.trim();

  return {
    name: draft.name.trim(),
    gameType: draft.gameType,
    platform: draft.platform,
    description: draft.description.trim() || "A new game asset workspace.",
    reference: draft.reference.trim(),
    style: perspective,
    perspective,
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
  const hasKnownPerspective = isKnownOption(
    projectContextOptions.perspectives,
    project.perspective,
  );

  return {
    name: project.name,
    gameType: hasKnownGameType ? project.gameType : "Other",
    customGameType: hasKnownGameType ? "" : project.gameType,
    perspective: hasKnownPerspective ? project.perspective : "Other",
    customPerspective: hasKnownPerspective ? "" : project.perspective,
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
  const perspective = resolveEditableOption(
    draft.perspective,
    draft.customPerspective,
  );

  if (!name || !gameType || !perspective) return undefined;

  return {
    ...project,
    name,
    gameType,
    perspective,
    style: perspective,
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
