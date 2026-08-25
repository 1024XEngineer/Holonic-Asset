import { LoaderCircle, RotateCcw, Trash2 } from "lucide-react";
import { useTranslation } from "react-i18next";

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
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import {
  useDeleteGenerationRunMutation,
  useRetryGenerationRunMutation,
} from "@/model/generation";

export type GenerationRunRecoveryTarget = {
  projectId: string;
  runId: string;
};

export function GenerationRunRecoveryActions({
  name,
  target,
  variant = "queue",
}: {
  name: string;
  target: GenerationRunRecoveryTarget;
  variant?: "queue" | "dropdown";
}) {
  const { t } = useTranslation(["generation", "common"]);
  const retryMutation = useRetryGenerationRunMutation();
  const deleteMutation = useDeleteGenerationRunMutation();
  const isRetrying = retryMutation.isPending;
  const isDeleting = deleteMutation.isPending;
  const actionError = retryMutation.error ?? deleteMutation.error;
  const isDropdown = variant === "dropdown";

  return (
    <div
      className={cn(
        "flex flex-col gap-1",
        isDropdown ? "mt-2 items-start" : "items-end",
      )}
    >
      <div className="flex items-center gap-1">
        <Button
          type="button"
          variant="outline"
          size={isDropdown ? "xs" : "sm"}
          aria-label={t("retryTask", { name })}
          disabled={isRetrying || isDeleting}
          onClick={() => {
            deleteMutation.reset();
            retryMutation.mutate(target);
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
                size={isDropdown ? "xs" : "sm"}
                className="text-muted-foreground hover:bg-destructive/10 hover:text-destructive"
                aria-label={t("deleteTask", { name })}
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
                {t("deleteTaskConfirm", { name })}
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
                  deleteMutation.mutate(target);
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
          className={cn(
            "max-w-48 text-[11px] text-destructive",
            !isDropdown && "text-right",
          )}
          role="alert"
        >
          {t("taskActionError")}
        </p>
      ) : null}
    </div>
  );
}
