export const assetKeys = {
  all: ["assets"] as const,
  library: (userID: number, projectId: string) =>
    [...assetKeys.all, userID, "library", projectId] as const,
};
