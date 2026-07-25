import { useMutation, useQueryClient } from "@tanstack/react-query";

import { projectApi } from "./project.api";
import { assetKeys } from "@/api/asset/keys";
import { generationKeys } from "@/api/generation/keys";
import type { ProjectSummary } from "@/domain/project";
import { projectKeys } from "./keys";

export function useDeleteProjectMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: projectApi.delete,
    onSuccess: (_, projectId) => {
      queryClient.setQueryData<ProjectSummary[]>(
        projectKeys.list(),
        (current = []) => current.filter((project) => project.id !== projectId),
      );
      queryClient.removeQueries({ queryKey: assetKeys.library(projectId) });
      queryClient.removeQueries({ queryKey: generationKeys.runs(projectId) });
    },
  });
}
