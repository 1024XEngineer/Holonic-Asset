export type AudioTrackTone = "mint" | "gold" | "lunar" | "rose" | "blue";
export type AudioTrack = {
  id: string;
  name: string;
  offset: number;
  length: number;
  tone: AudioTrackTone;
  muted: boolean;
  loop: boolean;
};
export type AddAudioTrackInput = Omit<AudioTrack, "id">;
export type UpdateAudioTrackInput = {
  trackId: string;
  patch: Partial<Pick<AudioTrack, "offset" | "length" | "muted" | "loop">>;
};
export type GenerateAudioVariationInput = { prompt: string; duration: number };
