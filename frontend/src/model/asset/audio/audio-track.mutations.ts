import {
  mutationOptions,
  useMutation,
  useQueryClient,
  type QueryClient,
} from "@tanstack/react-query";

import type { AudioTrack } from "./types";
import { audioApi } from "./audio.api";
import { audioKeys } from "./audio.keys";
import { readAuthenticatedUserId } from "@/model/auth";

export function addAudioTrackMutationOptions(queryClient: QueryClient) {
  return mutationOptions({
    mutationFn: audioApi.addTrack,
    onSuccess: (track) => {
      const userID = readAuthenticatedUserId();
      queryClient.setQueryData<AudioTrack[]>(
        audioKeys.tracks(userID),
        (current = []) => [...current, track],
      );
    },
  });
}

export function updateAudioTrackMutationOptions(queryClient: QueryClient) {
  return mutationOptions({
    mutationFn: audioApi.updateTrack,
    onSuccess: (track) => {
      const userID = readAuthenticatedUserId();
      queryClient.setQueryData<AudioTrack[]>(
        audioKeys.tracks(userID),
        (current = []) =>
          current.map((item) => (item.id === track.id ? track : item)),
      );
    },
  });
}

export function deleteAudioTrackMutationOptions(queryClient: QueryClient) {
  return mutationOptions({
    mutationFn: audioApi.deleteTrack,
    onSuccess: (_, trackId) => {
      const userID = readAuthenticatedUserId();
      queryClient.setQueryData<AudioTrack[]>(
        audioKeys.tracks(userID),
        (current = []) => current.filter((track) => track.id !== trackId),
      );
    },
  });
}

export function generateAudioVariationMutationOptions(
  queryClient: QueryClient,
) {
  return mutationOptions({
    mutationFn: audioApi.generateVariation,
    onSuccess: (track) => {
      const userID = readAuthenticatedUserId();
      queryClient.setQueryData<AudioTrack[]>(
        audioKeys.tracks(userID),
        (current = []) => [...current, track],
      );
    },
  });
}

export function useAddAudioTrackMutation() {
  const queryClient = useAudioQueryClient();
  return useMutation(addAudioTrackMutationOptions(queryClient));
}

export function useUpdateAudioTrackMutation() {
  const queryClient = useAudioQueryClient();
  return useMutation(updateAudioTrackMutationOptions(queryClient));
}

export function useDeleteAudioTrackMutation() {
  const queryClient = useAudioQueryClient();
  return useMutation(deleteAudioTrackMutationOptions(queryClient));
}

export function useGenerateAudioVariationMutation() {
  const queryClient = useAudioQueryClient();
  return useMutation(generateAudioVariationMutationOptions(queryClient));
}

function useAudioQueryClient() {
  return useQueryClient();
}
