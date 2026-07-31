import { useQuery } from "@tanstack/react-query";

import { projectApi } from "./project.api";
import { projectKeys } from "./keys";

export function useProjectListQuery() {
  return useQuery({ queryKey: projectKeys.list(), queryFn: projectApi.list });
}
