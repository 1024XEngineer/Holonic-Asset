import { queryOptions, useQuery } from "@tanstack/react-query";

import { audioApi } from "./audio.api";
import { audioKeys } from "./audio.keys";

export function audioTracksQueryOptions() {
  return queryOptions({
    queryKey: audioKeys.tracks(),
    queryFn: audioApi.listTracks,
  });
}

export function useAudioTracksQuery() {
  return useQuery(audioTracksQueryOptions());
}
