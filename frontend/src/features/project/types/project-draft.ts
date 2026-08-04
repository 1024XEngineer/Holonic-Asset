export type NewProjectDraft = {
  name: string;
  gameType: string;
  platform: string;
  description: string;
  visualStyle: string;
  reference: string;
};

export type ProjectSettingsDraft = {
  name: string;
  gameType: string;
  customGameType: string;
  visualStyle: string;
  customVisualStyle: string;
  platform: string;
  description: string;
  visualDirection: string;
};
