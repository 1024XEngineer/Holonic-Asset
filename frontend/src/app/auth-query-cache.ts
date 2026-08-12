import type { QueryClient } from "@tanstack/react-query";

import { subscribeAuthSession } from "@/features/auth";

type AuthSessionSubscriber = typeof subscribeAuthSession;

export function synchronizeAuthQueryCache(
  queryClient: QueryClient,
  subscribe: AuthSessionSubscriber = subscribeAuthSession,
) {
  return subscribe(() => queryClient.clear());
}
