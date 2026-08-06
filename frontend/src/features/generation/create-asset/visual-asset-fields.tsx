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
import type { CreationRequest } from "@/model/generation";
import type { VisualAssetCreationDraft } from "../types";

export function VisualAssetFields({
  draft,
  onChange,
}: {
  draft: VisualAssetCreationDraft<File>;
  onChange: (draft: VisualAssetCreationDraft<File>) => void;
}) {
  const isSpriteAsset = draft.kind === "character" || draft.kind === "object";

  return (
    <>
      <div className="grid gap-4 sm:grid-cols-2">
        <label className="grid gap-2 text-sm font-medium">
          Canvas size
          <Input
            value={draft.canvasSize}
            onChange={(event) =>
              onChange({ ...draft, canvasSize: event.target.value })
            }
          />
        </label>
        <OptionSelect
          label="Perspective"
          value={draft.perspective}
          options={[
            ["top-down", "Top down"],
            ["side-on", "Side on"],
            ["isometric", "Isometric"],
          ]}
          onChange={(perspective) =>
            onChange({
              ...draft,
              perspective: perspective as NonNullable<
                CreationRequest<File>["perspective"]
              >,
            })
          }
        />
      </div>

      {isSpriteAsset ? (
        <>
          <OptionSelect
            label="Direction count"
            value={draft.directionCount}
            options={[
              ["1", "1 direction"],
              ["4", "4 directions"],
              ["8", "8 directions"],
            ]}
            onChange={(directionCount) =>
              onChange({
                ...draft,
                directionCount: directionCount as NonNullable<
                  CreationRequest<File>["directionCount"]
                >,
              })
            }
          />
          <div className="grid gap-2 text-sm font-medium">
            <span>Reference</span>
            <ImageDropzone
              fileName={draft.reference?.name}
              onSelect={(reference) => onChange({ ...draft, reference })}
              onClear={() => onChange({ ...draft, reference: undefined })}
            />
          </div>
        </>
      ) : null}
    </>
  );
}

function OptionSelect({
  label,
  options,
  value,
  onChange,
}: {
  label: string;
  options: [value: string, label: string][];
  value: string;
  onChange: (value: string) => void;
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
