import { ChevronDown } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

import type { SceneryAssetCreationDraft } from "../types";
import {
  defaultSceneryAspectRatio,
  getSceneryCanvasSize,
  sceneryAspectRatios,
  type SceneryAspectRatio,
} from "./scenery-aspect-ratio";

export function SceneryAssetFields({
  draft,
  onChange,
}: {
  draft: SceneryAssetCreationDraft;
  onChange: (draft: SceneryAssetCreationDraft) => void;
}) {
  const { t } = useTranslation("generation");
  const [open, setOpen] = useState(false);

  return (
    <div className="grid gap-2">
      <div>
        <p className="text-sm font-medium">{t("aspectRatio")}</p>
        <p className="mt-1 text-xs text-muted-foreground">
          {t("sceneryAspectRatioDescription")}
        </p>
      </div>
      <DropdownMenu modal={false} open={open} onOpenChange={setOpen}>
        <DropdownMenuTrigger
          render={
            <Button
              type="button"
              variant="outline"
              className="h-10 w-full justify-between px-3 font-normal"
              aria-label={t("aspectRatio")}
            />
          }
        >
          <span className="font-medium">{draft.aspectRatio}</span>
          <span className="ml-auto text-xs text-muted-foreground">
            {getSceneryCanvasSize(draft.aspectRatio)}
          </span>
          <ChevronDown className="size-4 text-muted-foreground" />
        </DropdownMenuTrigger>
        <DropdownMenuContent className="w-(--anchor-width) min-w-72">
          <DropdownMenuRadioGroup
            value={draft.aspectRatio}
            onValueChange={(value) => {
              const aspectRatio = value as SceneryAspectRatio;
              onChange({
                ...draft,
                aspectRatio,
                canvasSize: getSceneryCanvasSize(aspectRatio),
              });
              setOpen(false);
            }}
          >
            {sceneryAspectRatios.map((aspectRatio) => (
              <DropdownMenuRadioItem key={aspectRatio} value={aspectRatio}>
                <span className="font-medium">{aspectRatio}</span>
                {aspectRatio === defaultSceneryAspectRatio ? (
                  <span className="rounded-sm bg-muted px-1.5 py-0.5 text-[10px] font-medium text-muted-foreground">
                    {t("recommended")}
                  </span>
                ) : null}
                <span className="ml-auto pr-5 text-xs tabular-nums text-muted-foreground">
                  {getSceneryCanvasSize(aspectRatio)}
                </span>
              </DropdownMenuRadioItem>
            ))}
          </DropdownMenuRadioGroup>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}
