import { QueryClient } from "@tanstack/react-query";
import { describe, expect, it, vi } from "vitest";

import { synchronizeAuthQueryCache } from "./auth-query-cache";

describe("auth query cache synchronization", () => {
  it("clears cached data for every semantic session change", () => {
    const queryClient = createQueryClient();
    const subscription = createSubscription();
    synchronizeAuthQueryCache(queryClient, subscription.subscribe);

    subscription.notify();

    expect(queryClient.getQueryData(["projects"])).toBeUndefined();
  });

  it("unsubscribes from session changes during cleanup", () => {
    const queryClient = createQueryClient();
    const subscription = createSubscription();
    const unsubscribe = synchronizeAuthQueryCache(
      queryClient,
      subscription.subscribe,
    );

    unsubscribe();
    subscription.notify();

    expect(queryClient.getQueryData(["projects"])).toEqual([{ id: 1 }]);
    expect(subscription.unsubscribe).toHaveBeenCalledOnce();
  });
});

function createSubscription() {
  let listener: () => void = () => undefined;
  const unsubscribe = vi.fn(() => {
    listener = () => undefined;
  });
  return {
    notify: () => listener(),
    subscribe: (nextListener: () => void) => {
      listener = nextListener;
      return unsubscribe;
    },
    unsubscribe,
  };
}

function createQueryClient() {
  const queryClient = new QueryClient();
  queryClient.setQueryData(["projects"], [{ id: 1 }]);
  return queryClient;
}
