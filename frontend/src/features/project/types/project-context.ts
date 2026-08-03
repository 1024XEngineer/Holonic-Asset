import type { ProjectSummary } from "@/model/project/types";

import type {
  NewProjectDraft,
  ProjectSettingsDraft,
} from "./project-context.types";

export const projectContextOptions = {
  gameTypes: [
    "Role-playing game",
    "Platformer",
    "Puzzle",
    "Strategy",
    "Simulation",
  ],
  visualStyles: ["Pixel art", "Hand-painted", "Cartoon"],
  platforms: ["PC", "Mobile", "Web", "Console", "Multi-platform"],
} as const;

export const editableProjectContextOptions = {
  gameTypes: [...projectContextOptions.gameTypes, "Other"],
  visualStyles: [...projectContextOptions.visualStyles, "Other"],
} as const;

export function createNewProjectDraft(): NewProjectDraft {
  return {
    name: "",
    gameType: projectContextOptions.gameTypes[0],
    platform: projectContextOptions.platforms[0],
    description: "",
    visualStyle: projectContextOptions.visualStyles[0],
  };
}

export function toProjectSummary(
  draft: NewProjectDraft,
  fallbackId = `project-${Date.now()}`,
): ProjectSummary {
  const visualStyle = draft.visualStyle.trim();

  return {
    id: createProjectId(draft.name, fallbackId),
    name: draft.name.trim(),
    gameType: draft.gameType,
    platform: draft.platform,
    description: draft.description.trim() || "A new game asset workspace.",
    style: visualStyle,
    visualStyle,
    visualDirection: "",
    assetCount: 0,
  };
}

export function createProjectSettingsDraft(
  project: ProjectSummary,
): ProjectSettingsDraft {
  const hasKnownGameType = isKnownOption(
    projectContextOptions.gameTypes,
    project.gameType,
  );
  const hasKnownVisualStyle = isKnownOption(
    projectContextOptions.visualStyles,
    project.visualStyle,
  );

  return {
    name: project.name,
    gameType: hasKnownGameType ? project.gameType : "Other",
    customGameType: hasKnownGameType ? "" : project.gameType,
    visualStyle: hasKnownVisualStyle ? project.visualStyle : "Other",
    customVisualStyle: hasKnownVisualStyle ? "" : project.visualStyle,
    platform: project.platform,
    description: project.description,
    visualDirection: project.visualDirection,
  };
}

export function updateProjectSettingsDraft<
  K extends keyof ProjectSettingsDraft,
>(
  draft: ProjectSettingsDraft,
  key: K,
  value: ProjectSettingsDraft[K],
): ProjectSettingsDraft {
  const nextDraft = { ...draft, [key]: value } as ProjectSettingsDraft;

  if (key === "gameType" && value !== "Other") {
    return { ...nextDraft, customGameType: "" };
  }
  if (key === "visualStyle" && value !== "Other") {
    return { ...nextDraft, customVisualStyle: "" };
  }

  return nextDraft;
}

export function applyProjectSettings(
  project: ProjectSummary,
  draft: ProjectSettingsDraft,
): ProjectSummary | undefined {
  const name = draft.name.trim();
  const gameType = resolveEditableOption(draft.gameType, draft.customGameType);
  const visualStyle = resolveEditableOption(
    draft.visualStyle,
    draft.customVisualStyle,
  );

  if (!name || !gameType || !visualStyle) return undefined;

  return {
    ...project,
    name,
    gameType,
    visualStyle,
    style: visualStyle,
    platform: draft.platform,
    description: draft.description,
    visualDirection: draft.visualDirection,
  };
}

function createProjectId(name: string, fallbackId: string) {
  return (
    name
      .toLowerCase()
      .trim()
      .replace(/[^a-z0-9]+/g, "-")
      .replace(/(^-|-$)/g, "") || fallbackId
  );
}

function isKnownOption(options: readonly string[], value: string) {
  return options.includes(value);
}

function resolveEditableOption(value: string, customValue: string) {
  return value === "Other" ? customValue.trim() : value;
}
