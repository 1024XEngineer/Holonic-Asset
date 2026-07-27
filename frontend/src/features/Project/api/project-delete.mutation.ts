import { useMutation, useQueryClient } from "@tanstack/react-query";

import { projectApi } from "./project.api";
import type { ProjectSummary } from "../domain";
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
    },
  });
}
