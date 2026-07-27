import { QueryClientProvider } from "@tanstack/react-query";
import { RouterProvider } from "@tanstack/react-router";
import { useState } from "react";

import type { router as appRouter } from "@/app/router";
import { createQueryClient } from "@/app/query-client";

export function AppProviders({ router }: { router: typeof appRouter }) {
  const [queryClient] = useState(createQueryClient);

  return (
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  );
}
