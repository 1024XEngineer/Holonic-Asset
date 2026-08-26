import { coreApiClient, unwrapApiResponse } from "@/model/fetchers";

import type {
  CreateProjectTagRequest,
  CreateProjectTagResponse,
  DeleteProjectTagResponse,
  ListProjectTagsResponse,
  ProjectTagResponse,
  UpdateProjectTagRequest,
  UpdateProjectTagResponse,
} from "./tag.contract";

export type ProjectTag = {
  tagId: number;
  projectId: number;
  name: string;
  description: string;
  color: string;
};
export type ProjectTagInput = Pick<
  ProjectTag,
  "name" | "description" | "color"
>;

export type Tag = ProjectTag;
export type TagInput = ProjectTagInput;

export const tagApi = {
  list: async (projectId: string) => {
    const response = await listProjectTags(toProjectId(projectId));
    return response.tags.map(toProjectTag);
  },
  create: async (projectId: string, tag: TagInput) =>
    toProjectTag(
      (await createProjectTag(toProjectId(projectId), toRequest(tag))).tag,
    ),
  update: async (projectId: string, tagId: number, tag: Partial<TagInput>) =>
    toProjectTag(
      (
        await updateProjectTag(toProjectId(projectId), tagId, {
          ...(tag.name === undefined ? {} : { name: tag.name }),
          ...(tag.description === undefined
            ? {}
            : { description: tag.description }),
          ...(tag.color === undefined ? {} : { color: tag.color }),
        })
      ).tag,
    ),
  delete: async (projectId: string, tagId: number) => {
    await deleteProjectTag(toProjectId(projectId), tagId);
  },
};

export const projectTagApi = tagApi;

async function listProjectTags(projectId: number) {
  return unwrapApiResponse<ListProjectTagsResponse>(
    await coreApiClient.GET("/projects/{project_id}/tags", {
      params: { path: { project_id: projectId } },
      cache: "no-store",
    }),
  );
}

async function createProjectTag(
  projectId: number,
  request: CreateProjectTagRequest,
) {
  return unwrapApiResponse<CreateProjectTagResponse>(
    await coreApiClient.POST("/projects/{project_id}/tags", {
      params: { path: { project_id: projectId } },
      body: request,
    }),
  );
}

async function updateProjectTag(
  projectId: number,
  tagId: number,
  request: UpdateProjectTagRequest,
) {
  return unwrapApiResponse<UpdateProjectTagResponse>(
    await coreApiClient.PUT("/projects/{project_id}/tags/{tag_id}", {
      params: { path: { project_id: projectId, tag_id: tagId } },
      body: request,
    }),
  );
}

async function deleteProjectTag(projectId: number, tagId: number) {
  return unwrapApiResponse<DeleteProjectTagResponse>(
    await coreApiClient.DELETE("/projects/{project_id}/tags/{tag_id}", {
      params: { path: { project_id: projectId, tag_id: tagId } },
    }),
  );
}

function toProjectTag(response: ProjectTagResponse): Tag {
  const name = response.name.trim();
  if (!name) throw new Error("Project tag response has no name.");
  return {
    name,
    description: response.description?.trim() ?? "",
    color: /^#[0-9a-f]{6}$/i.test(response.color) ? response.color : "#4F46E5",
    tagId: response.tagId,
    projectId: response.projectId,
  };
}

function toRequest(tag: TagInput) {
  return {
    name: tag.name,
    ...(tag.description ? { description: tag.description } : {}),
    ...(tag.color ? { color: tag.color } : {}),
  };
}

function toProjectId(projectId: string) {
  const value = Number(projectId);
  if (!Number.isSafeInteger(value) || value <= 0) {
    throw new Error(
      "Project tag operations require a persisted Core API project.",
    );
  }
  return value;
}
