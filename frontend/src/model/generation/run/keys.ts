export const generationKeys = {
  all: ["generation"] as const,
  runs: (userID: number, projectId: string) =>
    [...generationKeys.all, userID, "runs", projectId] as const,
};
