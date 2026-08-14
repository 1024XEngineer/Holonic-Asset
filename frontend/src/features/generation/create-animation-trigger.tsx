import { Film, Gauge, LoaderCircle, Sparkles, Timer } from "lucide-react";
import { useState, type FormEvent } from "react";
import { useTranslation } from "react-i18next";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";
import {
  assetDirectionsByPerspective,
  generateAnimationRequestSchema,
  type GenerateAnimationRequest,
  type Perspective,
} from "@/model";

type CreateAnimationTriggerProps = {
  children: (openDialog: () => void) => React.ReactNode;
  isGenerating: boolean;
  perspective: Perspective;
  onGenerate: (request: GenerateAnimationRequest) => void;
};

const defaultRequest: GenerateAnimationRequest = {
  animationName: "",
  direction: "front",
  creativeBrief: "",
  frameCount: 8,
  fps: 12,
  duration: 5,
};

export function CreateAnimationTrigger({
  children,
  isGenerating,
  perspective,
  onGenerate,
}: CreateAnimationTriggerProps) {
  const { t } = useTranslation(["generation", "common"]);
  const [open, setOpen] = useState(false);
  const [request, setRequest] = useState(defaultRequest);
  const availableDirections = assetDirectionsByPerspective[perspective];

  const openDialog = () => {
    setRequest({
      ...defaultRequest,
      direction: availableDirections[0] ?? "front",
    });
    setOpen(true);
  };

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const result = generateAnimationRequestSchema.safeParse(request);
    if (!result.success || isGenerating) return;

    onGenerate(result.data);
    setOpen(false);
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      {children(openDialog)}
      <DialogContent className="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <span className="grid size-8 place-items-center rounded-lg bg-primary/10 text-primary">
              <Film className="size-4" />
            </span>
            {t("generateAnimation")}
          </DialogTitle>
          <DialogDescription>{t("animationDescription")}</DialogDescription>
        </DialogHeader>
        <form className="grid gap-6" onSubmit={handleSubmit}>
          <label
            className="grid gap-2 text-sm font-medium"
            htmlFor="animation-name"
          >
            {t("animationName")}
            <Input
              id="animation-name"
              autoFocus
              required
              placeholder={t("castSpellPlaceholder")}
              value={request.animationName}
              onChange={(event) =>
                setRequest((current) => ({
                  ...current,
                  animationName: event.target.value,
                }))
              }
            />
          </label>

          <fieldset className="grid gap-2">
            <legend className="text-sm font-medium">{t("direction")}</legend>
            <div
              className={cn(
                "grid overflow-hidden rounded-lg border bg-muted/40 p-1",
                availableDirections.length === 2
                  ? "grid-cols-2"
                  : "grid-cols-4",
              )}
            >
              {availableDirections.map((direction) => (
                <button
                  key={direction}
                  type="button"
                  aria-pressed={request.direction === direction}
                  onClick={() =>
                    setRequest((current) => ({ ...current, direction }))
                  }
                  className={cn(
                    "min-h-9 min-w-0 rounded-md px-2 text-xs font-medium transition-colors focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-ring",
                    request.direction === direction
                      ? "bg-background text-foreground shadow-sm ring-1 ring-foreground/10"
                      : "text-muted-foreground hover:bg-background/60 hover:text-foreground",
                  )}
                >
                  {t(`directions.${direction}`)}
                </button>
              ))}
            </div>
          </fieldset>

          <label
            className="grid gap-2 text-sm font-medium"
            htmlFor="generated-animation-prompt"
          >
            {t("creativeBrief")}
            <Textarea
              id="generated-animation-prompt"
              required
              className="min-h-24 resize-y"
              placeholder={t("motionPlaceholder")}
              value={request.creativeBrief}
              onChange={(event) =>
                setRequest((current) => ({
                  ...current,
                  creativeBrief: event.target.value,
                }))
              }
            />
          </label>

          <div className="grid gap-3 border-t pt-5 sm:grid-cols-3">
            <NumberField
              icon={<Film />}
              label={t("frameCount")}
              value={request.frameCount}
              min={1}
              max={32}
              onChange={(frameCount) =>
                setRequest((current) => ({ ...current, frameCount }))
              }
            />
            <NumberField
              icon={<Gauge />}
              label={t("fps")}
              value={request.fps}
              min={1}
              max={60}
              onChange={(fps) => setRequest((current) => ({ ...current, fps }))}
            />
            <NumberField
              icon={<Timer />}
              label={t("sourceDuration")}
              value={request.duration}
              min={4}
              max={15}
              suffix={t("secondsShort")}
              onChange={(duration) =>
                setRequest((current) => ({ ...current, duration }))
              }
            />
          </div>

          <p className="-mt-3 text-xs text-muted-foreground">
            {t("animationSummary", {
              frames: request.frameCount,
              fps: request.fps,
              seconds: request.duration,
            })}
          </p>

          <Button
            type="submit"
            className="w-full"
            disabled={
              isGenerating ||
              !generateAnimationRequestSchema.safeParse(request).success
            }
          >
            {isGenerating ? (
              <LoaderCircle className="animate-spin" />
            ) : (
              <Sparkles />
            )}
            {isGenerating ? t("queueingAnimation") : t("generateAnimation")}
          </Button>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function NumberField({
  icon,
  label,
  value,
  min,
  max,
  suffix,
  onChange,
}: {
  icon: React.ReactNode;
  label: string;
  value: number;
  min: number;
  max: number;
  suffix?: string;
  onChange: (value: number) => void;
}) {
  return (
    <label className="grid gap-2 text-sm font-medium">
      <span className="flex items-center gap-1.5 [&_svg]:size-3.5 [&_svg]:text-muted-foreground">
        {icon}
        {label}
      </span>
      <span className="relative">
        <Input
          type="number"
          inputMode="numeric"
          required
          min={min}
          max={max}
          value={value}
          className={suffix ? "pr-8" : undefined}
          onChange={(event) => {
            const nextValue = event.currentTarget.valueAsNumber;
            if (Number.isFinite(nextValue)) onChange(nextValue);
          }}
        />
        {suffix ? (
          <span className="pointer-events-none absolute inset-y-0 right-2.5 flex items-center text-xs text-muted-foreground">
            {suffix}
          </span>
        ) : null}
      </span>
      <span className="text-[11px] font-normal text-muted-foreground">
        {min}–{max}
      </span>
    </label>
  );
}
