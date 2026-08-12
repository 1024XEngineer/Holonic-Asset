import { useQuery } from "@tanstack/react-query";

import { projectApi } from "./project.api";
import { projectKeys } from "./keys";

export function useProjectListQuery(userID: number) {
  return useQuery({
    queryKey: projectKeys.list(),
    queryFn: () => projectApi.list(userID),
  });
}
