import { useMutation, useQueryClient } from "@tanstack/react-query";

import { projectApi } from "./project.api";
import type { ProjectSummary } from "../domain";
import { projectKeys } from "./keys";

export function useCreateProjectMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: projectApi.create,
    onSuccess: (project) => {
      queryClient.setQueryData<ProjectSummary[]>(
        projectKeys.list(),
        (current = []) => [...current, project],
      );
    },
  });
}
