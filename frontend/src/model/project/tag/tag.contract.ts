import type { components, operations } from "@/model/generated/core-api";

export type ProjectTagResponse = components["schemas"]["ProjectTagResponse"];
export type CreateProjectTagRequest =
  operations["createProjectTag"]["requestBody"]["content"]["application/json"];
export type CreateProjectTagResponse =
  operations["createProjectTag"]["responses"][200]["content"]["application/json"]["data"];
export type ListProjectTagsResponse =
  operations["listProjectTags"]["responses"][200]["content"]["application/json"]["data"];
export type UpdateProjectTagRequest =
  operations["updateProjectTag"]["requestBody"]["content"]["application/json"];
export type UpdateProjectTagResponse =
  operations["updateProjectTag"]["responses"][200]["content"]["application/json"]["data"];
export type DeleteProjectTagResponse =
  operations["deleteProjectTag"]["responses"][200]["content"]["application/json"]["data"];
