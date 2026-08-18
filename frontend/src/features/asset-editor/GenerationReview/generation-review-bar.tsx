import { Check, LoaderCircle, Sparkles, X } from "lucide-react";
import { useTranslation } from "react-i18next";

import { Button } from "@/components/ui/button";

export type GenerationReview = {
  name: string;
  prompt: string;
  pendingCount: number;
  isLoading: boolean;
  isUnavailable: boolean;
  isResolving: boolean;
  onApply: () => void;
  onDeny: () => void;
};

export function GenerationReviewBar({ review }: { review: GenerationReview }) {
  const { t } = useTranslation("editor");
  const message = review.isLoading
    ? t("loadingGenerationPreview")
    : review.isUnavailable
      ? t("generationPreviewUnavailable")
      : review.prompt;

  return (
    <section
      aria-label={t("generationReview")}
      className="flex min-h-16 shrink-0 items-center justify-between gap-3 border-t border-primary/20 bg-background px-3 py-2 sm:px-4"
    >
      <div className="flex min-w-0 items-center gap-3">
        <div className="grid size-8 shrink-0 place-items-center rounded-md bg-primary/10 text-primary">
          {review.isLoading ? (
            <LoaderCircle className="size-4 animate-spin" />
          ) : (
            <Sparkles className="size-4" />
          )}
        </div>
        <div className="min-w-0">
          <div className="flex min-w-0 items-center gap-2">
            <p className="truncate text-xs font-semibold">{review.name}</p>
            {review.pendingCount > 1 ? (
              <span className="shrink-0 text-[10px] text-muted-foreground">
                {t("generationReviewCount", { count: review.pendingCount })}
              </span>
            ) : null}
          </div>
          <p className="truncate text-[11px] text-muted-foreground">
            {message}
          </p>
        </div>
      </div>
      <div className="flex shrink-0 items-center gap-1.5">
        <Button
          size="sm"
          variant="outline"
          disabled={review.isResolving}
          onClick={review.onDeny}
        >
          <X data-icon="inline-start" />
          {t("denyGeneration")}
        </Button>
        <Button
          size="sm"
          disabled={
            review.isLoading || review.isUnavailable || review.isResolving
          }
          onClick={review.onApply}
        >
          {review.isResolving ? (
            <LoaderCircle className="animate-spin" data-icon="inline-start" />
          ) : (
            <Check data-icon="inline-start" />
          )}
          {t("applyGeneration")}
        </Button>
      </div>
    </section>
  );
}
