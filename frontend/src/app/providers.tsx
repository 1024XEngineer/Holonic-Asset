import { useEffect } from "react";
import { QueryClientProvider } from "@tanstack/react-query";
import { RouterProvider } from "@tanstack/react-router";
import { I18nextProvider } from "react-i18next";

import type { router as appRouter } from "@/app/router";
import { synchronizeAuthQueryCache } from "@/app/auth-query-cache";
import { queryClient } from "@/app/query-client";
import { i18n } from "@/i18n";

export function AppProviders({ router }: { router: typeof appRouter }) {
  useEffect(() => synchronizeAuthQueryCache(queryClient), []);

  return (
    <I18nextProvider i18n={i18n}>
      <QueryClientProvider client={queryClient}>
        <RouterProvider router={router} />
      </QueryClientProvider>
    </I18nextProvider>
  );
}
