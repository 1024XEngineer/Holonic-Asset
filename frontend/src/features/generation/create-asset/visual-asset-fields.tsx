import { ImageDropzone } from "@/components/ui/custom/image-dropzone";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ChevronDown } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { perspectiveOptions, type Perspective } from "@/model/project";
import type { VisualAssetCreationDraft } from "../types";

const perspectiveLabels: Record<Perspective, string> = {
  "Top-Down": "Top-Down",
  "Side-On": "Side-On",
  Isometric: "Isometric",
};

export function VisualAssetFields({
  draft,
  onChange,
}: {
  draft: VisualAssetCreationDraft<File>;
  onChange: (draft: VisualAssetCreationDraft<File>) => void;
}) {
  const { t } = useTranslation("generation");

  return (
    <>
      <div className="grid gap-4 sm:grid-cols-2">
        <label className="grid gap-2 text-sm font-medium">
          {t("canvasSize")}
          <Input
            value={draft.canvasSize}
            onChange={(event) =>
              onChange({ ...draft, canvasSize: event.target.value })
            }
          />
        </label>
        <OptionSelect
          label={t("perspective")}
          value={draft.perspective}
          options={perspectiveOptions.map((perspective) => [
            perspective,
            perspectiveLabels[perspective],
          ])}
          onChange={(perspective) => onChange({ ...draft, perspective })}
        />
      </div>

      <OptionSelect
        label={t("directionCount")}
        value={draft.directionCount}
        options={[
          ["1", t("directions", { count: 1 })],
          ["4", t("directions", { count: 4 })],
          ["8", t("directions", { count: 8 })],
        ]}
        onChange={(directionCount) =>
          onChange({
            ...draft,
            directionCount,
          })
        }
      />
      <div className="grid gap-2 text-sm font-medium">
        <span>{t("reference")}</span>
        <ImageDropzone
          fileName={draft.reference?.name}
          onSelect={(reference) => onChange({ ...draft, reference })}
          onClear={() => onChange({ ...draft, reference: undefined })}
        />
      </div>
    </>
  );
}

function OptionSelect<Value extends string>({
  label,
  options,
  value,
  onChange,
}: {
  label: string;
  options: readonly (readonly [value: Value, label: string])[];
  value: Value;
  onChange: (value: Value) => void;
}) {
  const [open, setOpen] = useState(false);
  const selectedLabel =
    options.find(([option]) => option === value)?.[1] ?? value;

  return (
    <label className="grid gap-2 text-sm font-medium">
      {label}
      <DropdownMenu modal={false} open={open} onOpenChange={setOpen}>
        <DropdownMenuTrigger
          render={
            <Button
              type="button"
              variant="outline"
              className="h-9 w-full justify-between px-3 font-normal"
            />
          }
        >
          {selectedLabel}
          <ChevronDown className="size-4 text-muted-foreground" />
        </DropdownMenuTrigger>
        <DropdownMenuContent className="w-(--anchor-width)">
          <DropdownMenuRadioGroup
            value={value}
            onValueChange={(nextValue) => {
              onChange(nextValue);
              setOpen(false);
            }}
          >
            {options.map(([option, optionLabel]) => (
              <DropdownMenuRadioItem key={option} value={option}>
                {optionLabel}
              </DropdownMenuRadioItem>
            ))}
          </DropdownMenuRadioGroup>
        </DropdownMenuContent>
      </DropdownMenu>
    </label>
  );
}
