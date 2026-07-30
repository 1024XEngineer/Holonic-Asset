import {
  addMockAudioTrack,
  deleteMockAudioTrack,
  generateMockAudioVariation,
  listMockAudioTracks,
  updateMockAudioTrack,
} from "./mock/audio";
import type {
  AddAudioTrackInput,
  AudioTrack,
  GenerateAudioVariationInput,
  UpdateAudioTrackInput,
} from "@/features/audio-studio/types";

export type AudioApi = {
  listTracks: () => Promise<AudioTrack[]>;
  addTrack: (input: AddAudioTrackInput) => Promise<AudioTrack>;
  updateTrack: (input: UpdateAudioTrackInput) => Promise<AudioTrack>;
  deleteTrack: (trackId: string) => Promise<void>;
  generateVariation: (
    input: GenerateAudioVariationInput,
  ) => Promise<AudioTrack>;
};

export const audioApi: AudioApi = {
  listTracks: listMockAudioTracks,
  addTrack: addMockAudioTrack,
  updateTrack: updateMockAudioTrack,
  deleteTrack: deleteMockAudioTrack,
  generateVariation: generateMockAudioVariation,
};
