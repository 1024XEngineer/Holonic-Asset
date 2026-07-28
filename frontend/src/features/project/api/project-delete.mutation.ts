import {
  mutationOptions,
  useMutation,
  useQueryClient,
  type QueryClient,
} from "@tanstack/react-query";

import { assetKeys } from "@/features/assets/api";
import { generationKeys } from "@/features/generation/api";
import { projectApi } from "./project.api";
import type { ProjectSummary } from "../domain";
import { projectKeys } from "./keys";

export function deleteProjectMutationOptions(queryClient: QueryClient) {
  return mutationOptions({
    mutationFn: projectApi.delete,
    onSuccess: (_, projectId) => {
      clearDeletedProjectCache(queryClient, projectId);
    },
  });
}

export function clearDeletedProjectCache(
  queryClient: QueryClient,
  projectId: string,
) {
  queryClient.setQueryData<ProjectSummary[]>(
    projectKeys.list(),
    (current = []) => current.filter((project) => project.id !== projectId),
  );
  queryClient.removeQueries({ queryKey: assetKeys.library(projectId) });
  queryClient.removeQueries({ queryKey: generationKeys.runs(projectId) });
}

export function useDeleteProjectMutation() {
  const queryClient = useQueryClient();
  return useMutation(deleteProjectMutationOptions(queryClient));
}
