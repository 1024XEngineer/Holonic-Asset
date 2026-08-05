export type NewProjectDraft = {
  name: string;
  gameType: string;
  platform: string;
  description: string;
  perspective: string;
  reference: string;
};

export type ProjectSettingsDraft = {
  name: string;
  gameType: string;
  customGameType: string;
  perspective: string;
  customPerspective: string;
  platform: string;
  description: string;
  visualDirection: string;
};
