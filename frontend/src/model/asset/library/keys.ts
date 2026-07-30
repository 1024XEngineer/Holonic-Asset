export const assetKeys = {
  all: ["assets"] as const,
  library: (projectId: string) =>
    [...assetKeys.all, "library", projectId] as const,
};
