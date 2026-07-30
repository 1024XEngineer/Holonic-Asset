export const recordKeys = {
  all: ["record"] as const,
  detail: (projectId: string, assetId: string) =>
    [...recordKeys.all, "detail", projectId, assetId] as const,
};
