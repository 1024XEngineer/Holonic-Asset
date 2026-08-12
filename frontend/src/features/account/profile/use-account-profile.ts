import { useSyncExternalStore } from "react";

import {
  readAccountProfile,
  subscribeAccountProfile,
} from "./account-profile.store";

export function useAccountProfile() {
  return useSyncExternalStore(
    subscribeAccountProfile,
    readAccountProfile,
    readAccountProfile,
  );
}
