import {
  addMockAudioTrack,
  deleteMockAudioTrack,
  generateMockAudioVariation,
  listMockAudioTracks,
  updateMockAudioTrack,
} from "@/api/mock";
import type {
  AddAudioTrackInput,
  AudioTrack,
  GenerateAudioVariationInput,
  UpdateAudioTrackInput,
} from "@/domain/audio";

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
