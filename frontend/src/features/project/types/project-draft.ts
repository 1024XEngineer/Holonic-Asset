import type { Perspective } from "@/model/project";

export type NewProjectDraft = {
  name: string;
  gameType: string;
  platform: string;
  description: string;
  perspective: Perspective;
  reference: string;
};

export type ProjectSettingsDraft = {
  name: string;
  gameType: string;
  customGameType: string;
  perspective: Perspective;
  platform: string;
  description: string;
  reference: string;
};
