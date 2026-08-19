export const generationKeys = {
  all: ["generation"] as const,
  candidate: (userID: number, runId: string) =>
    [...generationKeys.all, userID, "candidate", runId] as const,
  runs: (userID: number, projectId: string, assetId?: string) =>
    [
      ...generationKeys.all,
      userID,
      "runs",
      projectId,
      ...(assetId ? (["asset", assetId] as const) : []),
    ] as const,
};
