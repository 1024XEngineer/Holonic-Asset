import { useMutation, useQueryClient } from "@tanstack/react-query";

import { projectApi } from "./project.api";
import { projectKeys } from "./keys";
import { readAuthenticatedUserId } from "@/model/auth";

export function useCreateProjectMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: projectApi.create,
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: projectKeys.list(readAuthenticatedUserId()),
      });
    },
  });
}
