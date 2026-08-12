import { perspectiveOptions } from "@/model/project";
import type { CreateProjectInput, ProjectSummary } from "@/model/project";
import { perspectiveSchema } from "@/model/project";
import { z } from "zod";

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
  platforms: ["PC", "Mobile", "Web"],
} as const;

export const editableProjectContextOptions = {
  gameTypes: [...projectContextOptions.gameTypes, "Other"],
} as const;

const projectGameTypeSchema = z.enum(projectContextOptions.gameTypes);
const editableProjectGameTypeSchema = z.enum(
  editableProjectContextOptions.gameTypes,
);
const optionalEditableProjectGameTypeSchema = editableProjectGameTypeSchema.or(
  z.literal(""),
);
const projectPlatformSchema = z.enum(projectContextOptions.platforms);
const optionalProjectPlatformSchema = projectPlatformSchema.or(z.literal(""));
const projectNameSchema = z.string().trim().min(1, "Project name is required.");

const newProjectDraftSchema = z.object({
  name: projectNameSchema,
  gameType: projectGameTypeSchema,
  platform: projectPlatformSchema,
  description: z.string().trim(),
  perspective: perspectiveSchema,
  reference: z.string().trim(),
});

const projectSettingsDraftSchema = z
  .object({
    name: projectNameSchema,
    gameType: optionalEditableProjectGameTypeSchema,
    customGameType: z.string().trim(),
    perspective: perspectiveSchema,
    platform: optionalProjectPlatformSchema,
    description: z.string(),
    reference: z.string(),
  })
  .refine(
    ({ customGameType, gameType }) =>
      gameType !== "Other" || customGameType.length > 0,
    {
      error: "Custom game type is required.",
      path: ["customGameType"],
    },
  );

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
  draft: NewProjectDraft,
): CreateProjectInput {
  const value = newProjectDraftSchema.parse(draft);

  return {
    name: value.name,
    gameType: value.gameType,
    platform: value.platform,
    description: value.description || "A new game asset workspace.",
    reference: value.reference,
    style: value.perspective,
    perspective: value.perspective,
  };
}

export function toCreateBlankProjectInput(name: string): CreateProjectInput {
  return {
    name: projectNameSchema.parse(name),
    gameType: "",
    platform: "",
    description: "",
    perspective: projectContextOptions.perspectives[0],
    reference: "",
    style: "",
  };
}

export function createProjectSettingsDraft(
  project: ProjectSummary,
): ProjectSettingsDraft {
  const hasKnownGameType = projectGameTypeSchema.safeParse(
    project.gameType,
  ).success;
  const hasSelectableGameType = project.gameType === "" || hasKnownGameType;
  return {
    name: project.name,
    gameType: hasSelectableGameType ? project.gameType : "Other",
    customGameType: hasSelectableGameType ? "" : project.gameType,
    perspective: project.perspective,
    platform: project.platform,
    description: project.description,
    reference: project.reference,
  };
}

export function applyProjectSettings(
  project: ProjectSummary,
  draft: ProjectSettingsDraft,
): ProjectSummary | undefined {
  const result = projectSettingsDraftSchema.safeParse(draft);
  if (!result.success) return undefined;

  const value = result.data;
  const gameType =
    value.gameType === "Other" ? value.customGameType : value.gameType;

  return {
    ...project,
    name: value.name,
    gameType,
    perspective: value.perspective,
    style: value.perspective,
    platform: value.platform,
    description: value.description,
    reference: value.reference,
  };
}
