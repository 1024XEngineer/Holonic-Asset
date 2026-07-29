import type { ProjectSummary } from "@/model";

/** The Project shape currently returned by core-api. */
export type BackendProject = {
  userID: number;
  id: number;
  name: string;
  gameType: "RPG" | "ACT" | "SLG" | "Other";
  viewType: "TopDown" | "SideView" | "Isometric" | "Other";
  targetPlatform: "PC" | "Mobile" | "Web";
  description: string;
  reference: string;
  style: string;
};

export type CreateProjectRequest = Omit<BackendProject, "id">;
export type CreateProjectResponse = { id: number };
export type ListProjectsRequest = { userID: number };
export type ListProjectsResponse = { projects: BackendProject[] };
export type ProjectDetailRequest = { projectID: number };
export type ProjectDetailResponse = { project: BackendProject };
export type UpdateProjectRequest = {
  projectID: number;
  name?: string;
  gameType?: BackendProject["gameType"];
  viewType?: BackendProject["viewType"];
  targetPlatform?: BackendProject["targetPlatform"];
  description?: string;
  reference?: string;
  style?: string;
};
export type UpdateProjectResponse = { success: boolean };
export type DeleteProjectRequest = { projectID: number };
export type DeleteProjectResponse = { success: boolean };

/**
 * Field differences to resolve when connecting core-api:
 * - `id: string` (frontend) ↔ `id: number` (backend)
 * - `platform` ↔ `targetPlatform`; `visualStyle` ↔ `style`
 * - frontend-only: `visualDirection`, `assetCount`
 * - backend-only: `userID`, `viewType`, `reference`
 */
export declare function fromBackendProject(
  project: BackendProject,
): ProjectSummary;

export declare function toCreateProjectRequest(input: {
  project: ProjectSummary;
  userID: number;
  viewType: BackendProject["viewType"];
  reference: string;
}): CreateProjectRequest;

export declare function toUpdateProjectRequest(input: {
  project: ProjectSummary;
  viewType?: BackendProject["viewType"];
  reference?: string;
}): UpdateProjectRequest;

/**
 * UI-facing Project API. Mock and future HTTP implementations share this
 * interface; the conversion declarations above remain at the boundary.
 */
export type ProjectApi = {
  list: () => Promise<ProjectSummary[]>;
  detail: (projectId: string) => Promise<ProjectSummary>;
  create: (project: ProjectSummary) => Promise<ProjectSummary>;
  update: (project: ProjectSummary) => Promise<ProjectSummary>;
  delete: (projectId: string) => Promise<void>;
};
