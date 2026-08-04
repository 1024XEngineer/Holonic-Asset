export type Project = {
  id: string;
  name: string;
  style: string;
  gameType: string;
  platform: string;
  description: string;
};

export type ProjectSummary = Project & {
  visualStyle: string;
  visualDirection: string;
  assetCount: number;
};

export type CreateProjectInput = Omit<ProjectSummary, "id" | "assetCount">;
