import { useSyncExternalStore } from "react";

import { readAccountProfile, subscribeAccountProfile } from "@/model/account";

export function useAccountProfile() {
  return useSyncExternalStore(
    subscribeAccountProfile,
    readAccountProfile,
    readAccountProfile,
  );
}
