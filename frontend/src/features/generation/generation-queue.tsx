import { AlertCircle, LoaderCircle } from "lucide-react";
import { useTranslation } from "react-i18next";

import { Badge } from "@/components/ui/badge";
import { isGenerationRunActive, type GenerationRun } from "@/model/generation";

import { GenerationTaskList } from "./generation-task-list";

export function GenerationQueue({ runs }: { runs: GenerationRun[] }) {
  const { t } = useTranslation("generation");
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
          {t("queue")}
        </h2>
        <Badge variant="secondary">
          {runs.length} {t(runs.length === 1 ? "task_one" : "task_other")}
        </Badge>
      </div>
      <div className="mt-3 divide-y border-y" aria-live="polite">
        <GenerationTaskList tasks={runs} variant="queue" />
      </div>
    </section>
  );
}
