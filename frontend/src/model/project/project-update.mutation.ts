import { useMutation, useQueryClient } from "@tanstack/react-query";

import { projectApi } from "./project.api";
import type { ProjectSummary } from "./types";
import { projectKeys } from "./keys";

export function useUpdateProjectMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: projectApi.update,
    onSuccess: (project) => {
      queryClient.setQueryData<ProjectSummary[]>(
        projectKeys.list(),
        (current = []) =>
          current.map((item) => (item.id === project.id ? project : item)),
      );
    },
  });
}
