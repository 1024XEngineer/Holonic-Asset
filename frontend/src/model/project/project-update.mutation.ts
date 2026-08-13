import {
  useMutation,
  useQueryClient,
  type QueryClient,
} from "@tanstack/react-query";

import { projectApi } from "./project.api";
import type { ProjectSummary } from "./types";
import { projectKeys } from "./keys";
import { readAuthenticatedUserId } from "@/model/auth";

export function useUpdateProjectMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: projectApi.update,
    onSuccess: (project) => updateProjectCache(queryClient, project),
  });
}

export function updateProjectCache(
  queryClient: QueryClient,
  project: ProjectSummary,
  userID = readAuthenticatedUserId(),
) {
  queryClient.setQueryData<ProjectSummary[]>(
    projectKeys.list(userID),
    (current = []) =>
      current.map((item) => (item.id === project.id ? project : item)),
  );
  queryClient.setQueryData(projectKeys.detail(userID, project.id), project);
}
