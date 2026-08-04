import { Suspense } from "react";

import { cn } from "@/lib/utils";

export function ProjectChrome({
  children,
  sidebar,
}: {
  children: React.ReactNode;
  sidebar: React.ReactNode;
}) {
  return (
    <main
      className={cn(
        "flex w-full min-h-0 flex-1 overflow-hidden bg-muted/30",
        "h-[calc(100vh-3.5rem)]",
      )}
    >
      <Suspense
        fallback={<div className="w-16 shrink-0 border-r bg-sidebar md:w-80" />}
      >
        {sidebar}
      </Suspense>
      <section className="w-full min-h-0 min-w-0 flex-1 overflow-hidden">
        {children}
      </section>
    </main>
  );
}
