import { DataApiError } from "@/api/api-error";
import { runMockRequest, type MockRequestOptions } from "./request";
import { audioTrackSeed } from "./audio.seed";
import type {
  AddAudioTrackInput,
  AudioTrack,
  AudioTrackTone,
  GenerateAudioVariationInput,
  UpdateAudioTrackInput,
} from "@/domain/audio";

const generatedTrackTones: AudioTrackTone[] = [
  "blue",
  "rose",
  "mint",
  "gold",
  "lunar",
];

let tracks = createSeedTracks();

export function listMockAudioTracks(options?: MockRequestOptions) {
  return runMockRequest(() => cloneTracks(tracks), options);
}

export function addMockAudioTrack(
  input: AddAudioTrackInput,
  options?: MockRequestOptions,
) {
  return runMockRequest(() => {
    validateTrackInput(input);

    const track: AudioTrack = {
      ...input,
      name: input.name.trim(),
      id: `audio-${crypto.randomUUID()}`,
    };
    tracks = [...tracks, track];
    return structuredClone(track);
  }, options);
}

export function updateMockAudioTrack(
  input: UpdateAudioTrackInput,
  options?: MockRequestOptions,
) {
  return runMockRequest(() => {
    const track = findTrack(input.trackId);
    const updatedTrack = { ...track, ...input.patch };
    validateTrackInput(updatedTrack);

    tracks = tracks.map((item) =>
      item.id === input.trackId ? updatedTrack : item,
    );
    return structuredClone(updatedTrack);
  }, options);
}

export function deleteMockAudioTrack(
  trackId: string,
  options?: MockRequestOptions,
) {
  return runMockRequest(() => {
    findTrack(trackId);
    tracks = tracks.filter((track) => track.id !== trackId);
  }, options);
}

export function generateMockAudioVariation(
  input: GenerateAudioVariationInput,
  options?: MockRequestOptions,
) {
  return runMockRequest(() => {
    const prompt = input.prompt.trim();
    if (!prompt) {
      throw new DataApiError(
        "BAD_REQUEST",
        "An audio generation prompt is required.",
      );
    }
    if (!Number.isFinite(input.duration) || input.duration <= 0) {
      throw new DataApiError(
        "BAD_REQUEST",
        "Audio duration must be a positive finite number.",
        { duration: input.duration },
      );
    }
    if (input.duration > 90) {
      throw new DataApiError(
        "BAD_REQUEST",
        "Audio duration cannot exceed the 90 second timeline.",
        { duration: input.duration },
      );
    }

    const generatedCount = tracks.filter((track) =>
      track.id.startsWith("generated-audio-"),
    ).length;
    const offset = Math.max(
      0,
      Math.min(70, 90 - input.duration, tracks.length * 14),
    );
    const track: AudioTrack = {
      id: `generated-audio-${crypto.randomUUID()}`,
      name: `generated-variation-${generatedCount + 1}.wav`,
      offset,
      length: input.duration,
      tone: generatedTrackTones[generatedCount % generatedTrackTones.length],
      muted: false,
      loop: false,
    };
    tracks = [...tracks, track];
    return structuredClone(track);
  }, options);
}

export function resetAudioMockData() {
  tracks = createSeedTracks();
}

function createSeedTracks(): AudioTrack[] {
  return audioTrackSeed.map((track) => ({ ...track }));
}

function cloneTracks(value: AudioTrack[]) {
  return structuredClone(value);
}

function findTrack(trackId: string) {
  const track = tracks.find((item) => item.id === trackId);
  if (!track) {
    throw new DataApiError("NOT_FOUND", "Audio track was not found.", {
      trackId,
    });
  }
  return track;
}

function validateTrackInput(input: AddAudioTrackInput) {
  if (!input.name.trim()) {
    throw new DataApiError("BAD_REQUEST", "Audio track name is required.");
  }
  if (!Number.isFinite(input.offset) || input.offset < 0) {
    throw new DataApiError(
      "BAD_REQUEST",
      "Audio track offset must be a non-negative finite number.",
      { offset: input.offset },
    );
  }
  if (!Number.isFinite(input.length) || input.length <= 0) {
    throw new DataApiError(
      "BAD_REQUEST",
      "Audio track length must be a positive finite number.",
      { length: input.length },
    );
  }
}
