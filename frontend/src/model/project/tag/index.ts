export {
  projectTagApi,
  tagApi,
  type ProjectTag,
  type ProjectTagInput,
  type Tag,
  type TagInput,
} from "./tag.api";
export {
  type CreateProjectTagRequest,
  type CreateProjectTagResponse,
  type DeleteProjectTagResponse,
  type ListProjectTagsResponse,
  type ProjectTagResponse,
  type UpdateProjectTagRequest,
  type UpdateProjectTagResponse,
} from "./tag.contract";
export { projectTagKeys, tagKeys } from "./tag.keys";
export {
  useCreateProjectTagMutation,
  useCreateTagMutation,
  useDeleteProjectTagMutation,
  useDeleteTagMutation,
  useUpdateProjectTagMutation,
  useUpdateTagMutation,
} from "./tag.mutations";
export { useProjectTagsQuery, useTagsQuery } from "./tags.query";
