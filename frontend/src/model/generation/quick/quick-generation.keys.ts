export const quickGenerationKeys = {
  all: ["quick-generation"] as const,
  assets: (userID: number) =>
    [...quickGenerationKeys.all, userID, "assets"] as const,
};
