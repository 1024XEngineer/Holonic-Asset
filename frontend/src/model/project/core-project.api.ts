import type {
  CreateProjectRequest,
  CreateProjectResponse,
  DeleteProjectRequest,
  DeleteProjectResponse,
  ListProjectsResponse,
  ProjectDetailResponse,
  UpdateProjectRequest,
  UpdateProjectResponse,
} from "./project.contract";
import { getEnvelope, postEnvelope } from "@/model/fetchers";

export const coreProjectApi = {
  create: (request: CreateProjectRequest) =>
    postEnvelope<CreateProjectResponse>("/project/create", request),
  list: (userID: number) =>
    getEnvelope<ListProjectsResponse>("/project/list", { userID }),
  detail: (projectID: number) =>
    getEnvelope<ProjectDetailResponse>("/project/detail", { projectID }),
  update: (request: UpdateProjectRequest) =>
    postEnvelope<UpdateProjectResponse>("/project/update", request),
  delete: (request: DeleteProjectRequest) =>
    postEnvelope<DeleteProjectResponse>("/project/delete", request),
};
