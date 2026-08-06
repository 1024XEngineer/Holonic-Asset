import type { components, operations } from "@/model/generated/core-api";
import type { Perspective } from "./types";

type Schemas = components["schemas"];

export type ProjectResponse = Schemas["ProjectResponse"];
export type ProjectGameType = ProjectResponse["gameType"];
export type ProjectPerspective = Perspective;
export type ProjectPlatform = ProjectResponse["targetPlatform"];
export type CreateProjectRequest =
  operations["createProject"]["requestBody"]["content"]["application/json"];
export type CreateProjectResponse =
  operations["createProject"]["responses"][200]["content"]["application/json"]["data"];
export type ListProjectsResponse =
  operations["listProjects"]["responses"][200]["content"]["application/json"]["data"];
export type ProjectDetailResponse =
  operations["getProject"]["responses"][200]["content"]["application/json"]["data"];
export type UpdateProjectRequest =
  operations["updateProject"]["requestBody"]["content"]["application/json"];
export type UpdateProjectResponse =
  operations["updateProject"]["responses"][200]["content"]["application/json"]["data"];
export type DeleteProjectRequest =
  operations["deleteProject"]["requestBody"]["content"]["application/json"];
export type DeleteProjectResponse =
  operations["deleteProject"]["responses"][200]["content"]["application/json"]["data"];
