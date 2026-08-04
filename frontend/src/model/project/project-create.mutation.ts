import { useMutation, useQueryClient } from "@tanstack/react-query";

import { projectApi } from "./project.api";
import type { ProjectSummary } from "./types";
import { projectKeys } from "./keys";

export function useCreateProjectMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: projectApi.create,
    onSuccess: (project) => {
      const current = queryClient.getQueryData<ProjectSummary[]>(
        projectKeys.list(),
      );
      if (!current) return;

      queryClient.setQueryData<ProjectSummary[]>(projectKeys.list(), [
        ...current,
        project,
      ]);
    },
  });
}
