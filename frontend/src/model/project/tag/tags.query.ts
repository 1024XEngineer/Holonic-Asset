import { useQuery } from "@tanstack/react-query";

import { tagApi } from "./tag.api";
import { tagKeys } from "./tag.keys";

export function useTagsQuery(projectId: string | undefined) {
  return useQuery({
    queryKey: tagKeys.list(projectId ?? "unselected"),
    queryFn: () => tagApi.list(projectId!),
    enabled: Boolean(projectId),
  });
}

export const useProjectTagsQuery = useTagsQuery;
