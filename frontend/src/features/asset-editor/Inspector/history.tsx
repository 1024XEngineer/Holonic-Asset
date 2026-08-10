import { History } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import type { AssetRevision } from "@/model";

export function InspectorHistory({ entries }: { entries: AssetRevision[] }) {
  if (entries.length === 0) {
    return (
      <div className="py-8 text-center text-xs text-muted-foreground">
        <History className="mx-auto mb-2 size-6" />
        No saved records
      </div>
    );
  }

  return (
    <ol className="space-y-0">
      {entries.map((entry, index) => (
        <li
          key={entry.id}
          className={`relative border-l pl-4 ${index === entries.length - 1 ? "pb-1" : "pb-5"}`}
        >
          <span className="absolute top-1.5 -left-[5px] size-2 rounded-full border-2 border-background bg-muted-foreground" />
          <div className="flex items-center justify-between gap-2">
            <span className="font-mono text-[11px] text-muted-foreground">
              {entry.version}
            </span>
            {entry.isCurrent ? (
              <Badge variant="secondary">Current</Badge>
            ) : null}
          </div>
          <p className="mt-2 text-xs font-medium leading-5">
            {entry.description}
          </p>
        </li>
      ))}
    </ol>
  );
}
