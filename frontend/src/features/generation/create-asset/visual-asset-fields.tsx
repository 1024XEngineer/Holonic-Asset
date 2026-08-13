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
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { assetCanvasSizeOptions } from "@/model/asset";
import { ChevronDown } from "lucide-react";
import type React from "react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { perspectiveOptions, type Perspective } from "@/model/project";
import type { VisualAssetCreationDraft } from "../types";

const perspectiveLabels: Record<Perspective, string> = {
  "Top-Down": "Top-Down",
  "Side-On": "Side-On",
  Isometric: "Isometric",
};

const otherCanvasSize = "other";

function positiveDigits(value: string) {
  return value.replace(/\D/g, "").replace(/^0+/, "");
}

export function VisualAssetFields({
  draft,
  onChange,
}: {
  draft: VisualAssetCreationDraft<File>;
  onChange: (draft: VisualAssetCreationDraft<File>) => void;
}) {
  const { t } = useTranslation("generation");
  const isPreset = assetCanvasSizeOptions.includes(
    draft.canvasSize as (typeof assetCanvasSizeOptions)[number],
  );
  const [selection, setSelection] = useState(
    isPreset ? draft.canvasSize : otherCanvasSize,
  );
  const [width, setWidth] = useState("");
  const [height, setHeight] = useState("");

  const updateCustomSize = (nextWidth: string, nextHeight: string) => {
    setWidth(nextWidth);
    setHeight(nextHeight);
    if (nextWidth && nextHeight) {
      onChange({ ...draft, canvasSize: `${nextWidth} × ${nextHeight} px` });
    } else if (draft.canvasSize) {
      onChange({ ...draft, canvasSize: "" });
    }
  };

  return (
    <>
      <div className="grid gap-4 sm:grid-cols-2">
        <DropdownField
          label={t("canvasSize")}
          value={
            selection === otherCanvasSize ? t("canvasSizeOther") : selection
          }
          options={[...assetCanvasSizeOptions, otherCanvasSize]}
          optionLabels={{ [otherCanvasSize]: t("canvasSizeOther") }}
          onChange={(value) => {
            setSelection(value);
            if (value !== otherCanvasSize) {
              onChange({ ...draft, canvasSize: value });
              return;
            }
            setWidth("");
            setHeight("");
            onChange({ ...draft, canvasSize: "" });
          }}
        />
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

      {selection === otherCanvasSize ? (
        <CustomCanvasSizeField
          width={width}
          height={height}
          onChange={updateCustomSize}
        />
      ) : null}

      <div className="grid gap-2 text-sm font-medium">
        <span>{t("reference")}</span>
        <ImageDropzone
          value={draft.reference}
          onChange={(reference) => onChange({ ...draft, reference })}
        />
      </div>
    </>
  );
}

function CustomCanvasSizeField({
  width,
  height,
  onChange,
}: {
  width: string;
  height: string;
  onChange: (width: string, height: string) => void;
}) {
  const { t } = useTranslation("generation");

  return (
    <label className="grid gap-2 text-sm font-medium">
      {t("customCanvasSize")}
      <div className="grid grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)_auto] items-center gap-2">
        <DimensionInput
          aria-label={t("canvasWidth")}
          placeholder={t("canvasWidthPlaceholder")}
          value={width}
          onChange={(event) =>
            onChange(positiveDigits(event.target.value), height)
          }
          message={t("canvasSizeInvalid")}
        />
        <span aria-hidden="true">×</span>
        <DimensionInput
          aria-label={t("canvasHeight")}
          placeholder={t("canvasHeightPlaceholder")}
          value={height}
          onChange={(event) =>
            onChange(width, positiveDigits(event.target.value))
          }
          message={t("canvasSizeInvalid")}
        />
        <span className="text-muted-foreground">px</span>
      </div>
    </label>
  );
}

function DimensionInput({
  message,
  ...props
}: React.ComponentProps<typeof Input> & { message: string }) {
  const [focused, setFocused] = useState(false);
  const [invalidAttempt, setInvalidAttempt] = useState(false);
  const invalid = focused && invalidAttempt && !props.value;

  return (
    <Tooltip open={invalid}>
      <TooltipTrigger
        render={
          <Input
            {...props}
            inputMode="numeric"
            min="1"
            pattern="[0-9]*"
            onChange={(event) => {
              const rawValue = event.target.value;
              const nextValue = positiveDigits(rawValue);
              setInvalidAttempt(rawValue.length > 0 && nextValue.length === 0);
              props.onChange?.(event);
            }}
            onFocus={(event) => {
              setFocused(true);
              props.onFocus?.(event);
            }}
            onBlur={(event) => {
              setFocused(false);
              props.onBlur?.(event);
            }}
            aria-invalid={invalid}
          />
        }
      />
      <TooltipContent side="bottom">{message}</TooltipContent>
    </Tooltip>
  );
}

function DropdownField({
  label,
  value,
  options,
  optionLabels = {},
  onChange,
}: {
  label: string;
  value: string;
  options: readonly string[];
  optionLabels?: Record<string, string>;
  onChange: (value: string) => void;
}) {
  const [open, setOpen] = useState(false);

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
          {value}
          <ChevronDown className="size-4 text-muted-foreground" />
        </DropdownMenuTrigger>
        <DropdownMenuContent className="w-(--anchor-width)">
          <DropdownMenuRadioGroup
            value={
              options.find((option) => optionLabels[option] === value) ?? value
            }
            onValueChange={(nextValue) => {
              onChange(nextValue);
              setOpen(false);
            }}
          >
            {options.map((option) => (
              <DropdownMenuRadioItem key={option} value={option}>
                {optionLabels[option] ?? option}
              </DropdownMenuRadioItem>
            ))}
          </DropdownMenuRadioGroup>
        </DropdownMenuContent>
      </DropdownMenu>
    </label>
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
