import { AlertCircle, LoaderCircle } from "lucide-react";
import { useTranslation } from "react-i18next";

import { AssetKindIcon, getAssetKindConfig } from "@/components/asset-kind";
import { Badge } from "@/components/ui/badge";
import { isGenerationRunActive, type GenerationRun } from "@/model/generation";

export function GenerationQueue({ runs }: { runs: GenerationRun[] }) {
  const { t } = useTranslation("workspace");
  if (runs.length === 0) return null;
  const hasActiveRuns = runs.some(isGenerationRunActive);

  return (
    <section className="border-b py-5" aria-labelledby="generation-queue-title">
      <div className="flex items-center justify-between gap-3">
        <h2
          id="generation-queue-title"
          className="flex items-center gap-2 text-sm font-semibold"
        >
          {hasActiveRuns ? (
            <LoaderCircle className="size-4 animate-spin text-muted-foreground" />
          ) : (
            <AlertCircle className="size-4 text-destructive" />
          )}
          {t("generation.queue")}
        </h2>
        <Badge variant="secondary">
          {runs.length}{" "}
          {t(
            runs.length === 1 ? "generation.task_one" : "generation.task_other",
          )}
        </Badge>
      </div>
      <div className="mt-3 divide-y border-y" aria-live="polite">
        {runs.map((run) => {
          const isFailed = run.status === "failed";

          return (
            <div key={run.id} className="flex min-w-0 items-center gap-3 py-3">
              <span className="grid size-8 shrink-0 place-items-center rounded-md bg-muted text-muted-foreground">
                <AssetKindIcon kind={run.kind} className="size-4" />
              </span>
              <div className="min-w-0 flex-1">
                <p className="truncate text-sm font-medium">{run.name}</p>
                <p className="truncate text-xs text-muted-foreground">
                  {run.prompt}
                </p>
              </div>
              <span className="hidden shrink-0 text-xs text-muted-foreground md:block">
                {getAssetKindConfig(run.kind).label}
              </span>
              <Badge variant={isFailed ? "destructive" : "outline"}>
                {isFailed ? (
                  <AlertCircle />
                ) : (
                  <LoaderCircle className="animate-spin" />
                )}
                {run.status === "failed" ? t("generation.failed") : run.status}
              </Badge>
            </div>
          );
        })}
      </div>
    </section>
  );
}
