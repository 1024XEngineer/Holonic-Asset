export const projectKeys = {
  all: ["projects"] as const,
  list: (userID: number) => [...projectKeys.all, userID, "list"] as const,
  detail: (userID: number, projectId: string) =>
    [...projectKeys.all, userID, "detail", projectId] as const,
};
