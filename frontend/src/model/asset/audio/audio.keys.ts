export const audioKeys = {
  all: ["audio"] as const,
  tracks: (userID: number) => [...audioKeys.all, userID, "tracks"] as const,
};
