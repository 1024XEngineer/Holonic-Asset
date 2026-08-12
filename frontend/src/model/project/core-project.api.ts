import type {
  CreateProjectRequest,
  CreateProjectResponse,
  DeleteProjectRequest,
  DeleteProjectResponse,
  GenerateProjectReferenceRequest,
  GenerateProjectReferenceResponse,
  ListProjectsResponse,
  ProjectDetailResponse,
  UpdateProjectRequest,
  UpdateProjectResponse,
} from "./project.contract";
import {
  deleteEnvelope,
  getEnvelope,
  postEnvelope,
  putEnvelope,
} from "@/model/fetchers";

export const coreProjectApi = {
  create: (request: CreateProjectRequest) =>
    postEnvelope<CreateProjectResponse>("/project/create", request),
  generateReference: (request: GenerateProjectReferenceRequest) =>
    postEnvelope<GenerateProjectReferenceResponse>(
      "/project/reference/generate",
      request,
    ),
  list: (userID: number) =>
    getEnvelope<ListProjectsResponse>("/project/list", { userID }),
  detail: (projectID: number) =>
    getEnvelope<ProjectDetailResponse>("/project/detail", { projectID }),
  update: (request: UpdateProjectRequest) =>
    putEnvelope<UpdateProjectResponse>("/project/update", request),
  delete: (request: DeleteProjectRequest) =>
    deleteEnvelope<DeleteProjectResponse>("/project/delete", request),
};
