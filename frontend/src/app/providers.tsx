import { QueryClientProvider } from "@tanstack/react-query";
import { RouterProvider } from "@tanstack/react-router";

import type { router as appRouter } from "@/app/router";
import { queryClient } from "@/app/query-client";

export function AppProviders({ router }: { router: typeof appRouter }) {
  return (
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  );
}
