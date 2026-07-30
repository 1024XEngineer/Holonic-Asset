export const quickGenerationKeys = {
  all: ["quick-generation"] as const,
  assets: () => [...quickGenerationKeys.all, "assets"] as const,
};
