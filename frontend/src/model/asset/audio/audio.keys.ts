export const audioKeys = {
  all: ["audio"] as const,
  tracks: () => [...audioKeys.all, "tracks"] as const,
};
