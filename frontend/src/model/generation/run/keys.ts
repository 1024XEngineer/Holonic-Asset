export const generationKeys = {
  all: ["generation"] as const,
  runs: (projectId: string) =>
    [...generationKeys.all, "runs", projectId] as const,
};
