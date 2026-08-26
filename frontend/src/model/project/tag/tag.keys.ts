export const tagKeys = {
  all: ["project-tags"] as const,
  list: (projectId: string) => [...tagKeys.all, projectId, "list"] as const,
};

export const projectTagKeys = tagKeys;
