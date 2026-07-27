import { create } from "zustand";
import { persist } from "zustand/middleware";

type PreferencesStore = {
  compact: boolean;
  notifications: boolean;
  toggleCompact: () => void;
  toggleNotifications: () => void;
};

export const usePreferencesStore = create<PreferencesStore>()(
  persist(
    (set) => ({
      compact: false,
      notifications: true,
      toggleCompact: () => set((state) => ({ compact: !state.compact })),
      toggleNotifications: () =>
        set((state) => ({ notifications: !state.notifications })),
    }),
    { name: "game-asset-pack:preferences" },
  ),
);
