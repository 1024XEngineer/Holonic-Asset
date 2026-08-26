import { useMutation, useQueryClient } from "@tanstack/react-query";

import { readAuthenticatedUserId } from "@/model/auth";

import { tagApi, type TagInput } from "./tag.api";
import { tagKeys } from "./tag.keys";

export function useCreateTagMutation() {
  return useTagMutation((input: { projectId: string; tag: TagInput }) =>
    tagApi.create(input.projectId, input.tag),
  );
}

export function useUpdateTagMutation() {
  return useTagMutation(
    (input: { projectId: string; tagId: number; tag: Partial<TagInput> }) =>
      tagApi.update(input.projectId, input.tagId, input.tag),
  );
}

export function useDeleteTagMutation() {
  return useTagMutation((input: { projectId: string; tagId: number }) =>
    tagApi.delete(input.projectId, input.tagId),
  );
}

export const useCreateProjectTagMutation = useCreateTagMutation;
export const useUpdateProjectTagMutation = useUpdateTagMutation;
export const useDeleteProjectTagMutation = useDeleteTagMutation;

function useTagMutation<TInput, TResult>(
  mutationFn: (input: TInput & { projectId: string }) => Promise<TResult>,
) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn,
    onSuccess: (_result, { projectId }: TInput & { projectId: string }) => {
      void queryClient.invalidateQueries({
        queryKey: tagKeys.list(projectId),
      });
      void queryClient.invalidateQueries({
        queryKey: ["assets", readAuthenticatedUserId(), "library", projectId],
      });
    },
  });
}
