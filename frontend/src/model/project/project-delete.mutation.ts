import {
  mutationOptions,
  useMutation,
  useQueryClient,
  type QueryClient,
} from "@tanstack/react-query";

import { assetKeys } from "../asset/library/keys";
import { generationKeys } from "../generation/run/keys";
import { projectApi } from "./project.api";
import type { ProjectSummary } from "./types";
import { projectKeys } from "./keys";
import { readAuthenticatedUserId } from "@/model/auth";

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
  userID = readAuthenticatedUserId(),
) {
  queryClient.setQueryData<ProjectSummary[]>(
    projectKeys.list(userID),
    (current = []) => current.filter((project) => project.id !== projectId),
  );
  queryClient.removeQueries({
    queryKey: projectKeys.detail(userID, projectId),
  });
  queryClient.removeQueries({ queryKey: assetKeys.library(userID, projectId) });
  queryClient.removeQueries({
    queryKey: generationKeys.runs(userID, projectId),
  });
}

export function useDeleteProjectMutation() {
  const queryClient = useQueryClient();
  return useMutation(deleteProjectMutationOptions(queryClient));
}
