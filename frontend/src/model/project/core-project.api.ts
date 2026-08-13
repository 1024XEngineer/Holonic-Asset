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
import { coreApiClient, unwrapApiResponse } from "@/model/fetchers";

export const coreProjectApi = {
  create: async (request: CreateProjectRequest) =>
    unwrapApiResponse<CreateProjectResponse>(
      await coreApiClient.POST("/project/create", { body: request }),
    ),
  generateReference: async (request: GenerateProjectReferenceRequest) =>
    unwrapApiResponse<GenerateProjectReferenceResponse>(
      await coreApiClient.POST("/project/reference/generate", {
        body: request,
      }),
    ),
  list: async (userID: number) =>
    unwrapApiResponse<ListProjectsResponse>(
      await coreApiClient.GET("/project/list", {
        params: { query: { userID } },
      }),
    ),
  detail: async (projectID: number) =>
    unwrapApiResponse<ProjectDetailResponse>(
      await coreApiClient.GET("/project/detail", {
        params: { query: { projectID } },
      }),
    ),
  update: async (request: UpdateProjectRequest) =>
    unwrapApiResponse<UpdateProjectResponse>(
      await coreApiClient.PUT("/project/update", { body: request }),
    ),
  delete: async (request: DeleteProjectRequest) =>
    unwrapApiResponse<DeleteProjectResponse>(
      await coreApiClient.DELETE("/project/delete", { body: request }),
    ),
};
