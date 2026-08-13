export type AccountProfile = {
  name: string;
  email: string;
  avatarUrl?: string;
};

export const defaultAccountProfile: AccountProfile = {
  email: "demo@holonicasset.app",
  name: "Demo User",
};
