import type { Perspective } from "./perspective";

export type Project = {
  id: string;
  name: string;
  style: string;
  gameType: string;
  platform: string;
  description: string;
  reference: string;
};

export type ProjectSummary = Project & {
  perspective: Perspective;
  visualDirection: string;
  assetCount: number;
};

export type CreateProjectInput = Omit<ProjectSummary, "id" | "assetCount">;
