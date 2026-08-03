import { createRouter } from "@tanstack/react-router";

import { queryClient } from "@/app/query-client";
import { routeTree } from "@/app/routeTree.gen";

export const router = createRouter({
  routeTree,
  context: { queryClient },
  defaultPreload: "intent",
});

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}
