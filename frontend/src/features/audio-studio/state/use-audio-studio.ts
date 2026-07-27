import { useEffect, useState } from "react";

import {
  useAddAudioTrackMutation,
  useDeleteAudioTrackMutation,
  useGenerateAudioVariationMutation,
  useAudioTracksQuery,
  useUpdateAudioTrackMutation,
} from "../api";

export function useAudioStudio() {
  const tracksQuery = useAudioTracksQuery();
  const addTrackMutation = useAddAudioTrackMutation();
  const updateTrackMutation = useUpdateAudioTrackMutation();
  const deleteTrackMutation = useDeleteAudioTrackMutation();
  const generateVariationMutation = useGenerateAudioVariationMutation();
  const [playing, setPlaying] = useState(false);
  const [time, setTime] = useState(18);
  const [prompt, setPrompt] = useState(
    "Soft nighttime orchard ambience with distant insects and a warm melodic loop.",
  );
  const [duration, setDuration] = useState(30);
  const [masterMuted, setMasterMuted] = useState(false);
  const tracks = tracksQuery.data ?? [];
  const mutations = [
    addTrackMutation,
    updateTrackMutation,
    deleteTrackMutation,
    generateVariationMutation,
  ];
  const actionError = mutations.find((mutation) => mutation.error)?.error;
  const isMutating = mutations.some((mutation) => mutation.isPending);

  useEffect(() => {
    if (!playing) return;
    const timer = window.setInterval(
      () => setTime((value) => (value >= 90 ? 0 : value + 0.2)),
      200,
    );
    return () => window.clearInterval(timer);
  }, [playing]);

  function toggleTrack(id: string, key: "muted" | "loop") {
    const track = tracks.find((item) => item.id === id);
    if (!track) return;
    updateTrackMutation.mutate({
      trackId: id,
      patch: { [key]: !track[key] },
    });
  }

  function removeTrack(id: string) {
    deleteTrackMutation.mutate(id);
  }

  function addTrack() {
    addTrackMutation.mutate({
      name: "new-reference.mp3",
      offset: 10,
      length: 18,
      tone: "rose",
      muted: false,
      loop: false,
    });
  }

  function generateVariation() {
    if (!prompt.trim()) return;
    generateVariationMutation.mutate(
      { prompt, duration },
      { onSuccess: () => setPrompt("") },
    );
  }

  return {
    actionError,
    addTrack,
    duration,
    generateVariation,
    isLoading: tracksQuery.isPending,
    isMutating,
    loadError: tracksQuery.error,
    masterMuted,
    playing,
    prompt,
    reload: tracksQuery.refetch,
    removeTrack,
    setDuration,
    setPrompt,
    setTime,
    time,
    toggleMasterMuted: () => setMasterMuted((value) => !value),
    togglePlaying: () => setPlaying((value) => !value),
    toggleTrack,
    tracks,
  };
}
