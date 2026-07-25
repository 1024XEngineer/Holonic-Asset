import type { AudioTrack } from "@/types/audio";

export const audioTrackSeed = [
  {
    id: "orchard",
    name: "orchard-ambience.wav",
    offset: 0,
    length: 42,
    tone: "mint",
    muted: false,
    loop: true,
  },
  {
    id: "footsteps",
    name: "footsteps-grass.wav",
    offset: 24,
    length: 22,
    tone: "gold",
    muted: false,
    loop: false,
  },
  {
    id: "wind",
    name: "soft-wind.wav",
    offset: 56,
    length: 29,
    tone: "lunar",
    muted: false,
    loop: true,
  },
] as const satisfies readonly AudioTrack[];
