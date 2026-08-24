import type { ReactNode } from "react";

import { ScrollArea } from "@/components/ui/scroll-area";
import { cn } from "@/lib/utils";

export function AssetTree({
  title,
  description,
  count,
  children,
  emptyMessage,
  footer,
  contentClassName,
}: {
  title: string;
  description: string;
  count: number;
  children: ReactNode;
  emptyMessage?: string;
  footer?: ReactNode;
  contentClassName?: string;
}) {
  return (
    <aside className="flex w-full shrink-0 flex-col border-b bg-background lg:h-full lg:w-64 lg:border-b-0 lg:border-r">
      <div className="flex items-center justify-between border-b px-4 py-3">
        <div>
          <p className="text-xs font-semibold uppercase tracking-[0.14em] text-muted-foreground">
            {title}
          </p>
          <p className="mt-1 text-xs text-muted-foreground">{description}</p>
        </div>
        <span className="font-mono text-[10px] text-muted-foreground">
          {count}
        </span>
      </div>
      <ScrollArea className="max-h-56 flex-1 lg:max-h-none">
        <div className={cn("p-3", contentClassName)}>
          {emptyMessage ? (
            <div className="px-2 py-8 text-center text-xs text-muted-foreground">
              {emptyMessage}
            </div>
          ) : (
            children
          )}
        </div>
      </ScrollArea>
      {footer}
    </aside>
  );
}
