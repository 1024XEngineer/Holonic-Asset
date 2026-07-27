import { LoaderCircle } from "lucide-react";

import { Badge } from "@/components/ui/badge";

import type { GenerationRun } from "@/features/generation/domain";
import { AssetTypeIcon } from "@/features/assets/components";
import { getAssetTypeConfig } from "@/features/assets/components";

export function GenerationQueue({ runs }: { runs: GenerationRun[] }) {
  if (runs.length === 0) return null;

  return (
    <section className="mb-8" aria-labelledby="creation-queue-title">
      <div className="flex items-center justify-between gap-3">
        <div className="flex items-center gap-2">
          <LoaderCircle
            className="size-4 animate-spin text-muted-foreground"
            aria-hidden="true"
          />
          <h2 id="creation-queue-title" className="text-sm font-semibold">
            Creation queue
          </h2>
        </div>
        <Badge variant="secondary">{runs.length} active</Badge>
      </div>
      <div
        className="mt-3 divide-y overflow-hidden rounded-lg border bg-card"
        aria-live="polite"
      >
        {runs.map((item) => (
          <div
            key={item.id}
            className="flex min-w-0 items-center gap-3 px-3 py-2.5"
          >
            <div className="grid size-8 shrink-0 place-items-center rounded-md bg-muted text-muted-foreground">
              <AssetTypeIcon kind={item.kind} className="size-4" />
            </div>
            <div className="min-w-0 flex-1">
              <p className="truncate text-sm font-medium">{item.name}</p>
              <p className="truncate text-xs text-muted-foreground">
                {item.prompt}
              </p>
            </div>
            <div className="flex shrink-0 items-center gap-2">
              {item.kind !== "audio" ? (
                <span className="hidden text-xs text-muted-foreground sm:inline">
                  {item.canvasSize}
                </span>
              ) : null}
              <Badge variant="outline">
                {getAssetTypeConfig(item.kind).label}
              </Badge>
              <Badge
                variant={item.status === "failed" ? "destructive" : "secondary"}
              >
                {item.status}
              </Badge>
            </div>
          </div>
        ))}
      </div>
    </section>
  );
}
