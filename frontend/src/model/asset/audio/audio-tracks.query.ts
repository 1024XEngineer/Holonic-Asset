import { queryOptions, useQuery } from "@tanstack/react-query";

import { audioApi } from "./audio.api";
import { audioKeys } from "./audio.keys";
import { readAuthenticatedUserId } from "@/model/auth";

export function audioTracksQueryOptions() {
  const userID = readAuthenticatedUserId();
  return queryOptions({
    queryKey: audioKeys.tracks(userID),
    queryFn: audioApi.listTracks,
  });
}

export function useAudioTracksQuery() {
  return useQuery(audioTracksQueryOptions());
}
