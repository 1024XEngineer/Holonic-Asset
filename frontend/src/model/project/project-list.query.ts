import { useQuery } from "@tanstack/react-query";

import { projectApi } from "./project.api";
import { projectKeys } from "./keys";
import { readAuthenticatedUserId } from "@/model/auth";

export function useProjectListQuery() {
  const userID = readAuthenticatedUserId();
  return useQuery({
    queryKey: projectKeys.list(userID),
    queryFn: projectApi.list,
  });
}
