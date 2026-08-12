import { useSyncExternalStore } from "react";

import { readAuthSession, subscribeAuthSession } from "./auth-session.store";

export function useAuthSession() {
  return useSyncExternalStore(
    subscribeAuthSession,
    readAuthSession,
    readAuthSession,
  );
}
