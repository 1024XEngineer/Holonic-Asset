import { QueryClientProvider } from "@tanstack/react-query";
import { RouterProvider } from "@tanstack/react-router";

import type { router as appRouter } from "@/app/router";
import { queryClient } from "@/app/query-client";
import { Toaster } from "@/components/ui/toast";

export function AppProviders({ router }: { router: typeof appRouter }) {
  return (
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
      <Toaster />
    </QueryClientProvider>
  );
}
