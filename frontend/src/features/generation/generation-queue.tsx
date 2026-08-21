import { AlertCircle, LoaderCircle, RotateCcw, Trash2 } from "lucide-react";
import { useTranslation } from "react-i18next";

import { AssetKindIcon } from "@/components/asset-kind";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  isGenerationRunActive,
  type GenerationRun,
  useDeleteGenerationRunMutation,
  useRetryGenerationRunMutation,
} from "@/model/generation";

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
        {runs.map((run) => {
          const isFailed = run.status === "failed";

          return (
            <div
              key={run.id}
              className="flex min-w-0 flex-col gap-3 py-3 sm:flex-row sm:items-center"
            >
              <div className="flex min-w-0 flex-1 items-center gap-3">
                <span className="grid size-8 shrink-0 place-items-center rounded-md bg-muted text-muted-foreground">
                  <AssetKindIcon kind={run.kind} className="size-4" />
                </span>
                <div className="min-w-0 flex-1">
                  <p className="truncate text-sm font-medium">{run.name}</p>
                  <p className="truncate text-xs text-muted-foreground">
                    {run.prompt}
                  </p>
                </div>
              </div>
              <div className="flex shrink-0 items-center gap-2 self-end sm:self-auto">
                <Badge variant={isFailed ? "destructive" : "outline"}>
                  {isFailed ? (
                    <AlertCircle />
                  ) : (
                    <LoaderCircle className="animate-spin" />
                  )}
                  {t(`status.${run.status}`)}
                </Badge>
                {isFailed ? <FailedRunActions run={run} /> : null}
              </div>
            </div>
          );
        })}
      </div>
    </section>
  );
}

function FailedRunActions({ run }: { run: GenerationRun }) {
  const { t } = useTranslation(["generation", "common"]);
  const retryMutation = useRetryGenerationRunMutation();
  const deleteMutation = useDeleteGenerationRunMutation();
  const isRetrying = retryMutation.isPending;
  const isDeleting = deleteMutation.isPending;
  const actionError = retryMutation.error ?? deleteMutation.error;
  const input = { projectId: run.projectId, runId: run.id };

  return (
    <div className="flex flex-col items-end gap-1">
      <div className="flex items-center gap-1">
        <Button
          type="button"
          variant="outline"
          size="sm"
          aria-label={t("retryTask", { name: run.name })}
          disabled={isRetrying || isDeleting}
          onClick={() => {
            deleteMutation.reset();
            retryMutation.mutate(input);
          }}
        >
          {isRetrying ? (
            <LoaderCircle className="animate-spin" />
          ) : (
            <RotateCcw />
          )}
          {t("retry")}
        </Button>
        <AlertDialog>
          <AlertDialogTrigger
            render={
              <Button
                type="button"
                variant="ghost"
                size="sm"
                className="text-muted-foreground hover:bg-destructive/10 hover:text-destructive"
                aria-label={t("deleteTask", { name: run.name })}
                disabled={isRetrying || isDeleting}
              />
            }
          >
            {isDeleting ? (
              <LoaderCircle className="animate-spin" />
            ) : (
              <Trash2 />
            )}
            {t("delete")}
          </AlertDialogTrigger>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>
                {t("deleteTaskConfirm", { name: run.name })}
              </AlertDialogTitle>
              <AlertDialogDescription>
                {t("deleteTaskDescription")}
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>
                {t("common:actions.cancel")}
              </AlertDialogCancel>
              <AlertDialogAction
                variant="destructive"
                onClick={() => {
                  retryMutation.reset();
                  deleteMutation.mutate(input);
                }}
              >
                {t("delete")}
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      </div>
      {actionError ? (
        <p
          className="max-w-48 text-right text-[11px] text-destructive"
          role="alert"
        >
          {t("taskActionError")}
        </p>
      ) : null}
    </div>
  );
}
