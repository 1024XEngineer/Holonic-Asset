export const recordKeys = {
  all: ["record"] as const,
  detail: (userID: number, projectId: string, assetId: string) =>
    [...recordKeys.all, userID, "detail", projectId, assetId] as const,
};
