import { useMutation, useQueryClient } from "@tanstack/react-query";

import { projectApi } from "./project.api";
import { projectKeys } from "./keys";

export function useCreateProjectMutation(userID: number) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: Parameters<typeof projectApi.create>[1]) =>
      projectApi.create(userID, input),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: projectKeys.list() });
    },
  });
}
