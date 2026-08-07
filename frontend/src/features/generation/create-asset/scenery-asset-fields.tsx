import { ChevronDown } from "lucide-react";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import { ImageDropzone } from "@/components/ui/custom/image-dropzone";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import type { SceneryAssetCreationDraft } from "../types";

const itemCounts = [1, 2, 3, 4, 5, 6, 8];

export function SceneryAssetFields({
  draft,
  onChange,
}: {
  draft: SceneryAssetCreationDraft<File>;
  onChange: (draft: SceneryAssetCreationDraft<File>) => void;
}) {
  return (
    <>
      <label className="grid gap-2 text-sm font-medium">
        Style
        <Textarea
          required
          className="min-h-20 resize-none"
          placeholder="Describe the overall scene style..."
          value={draft.style}
          onChange={(event) =>
            onChange({ ...draft, style: event.target.value })
          }
        />
      </label>
      <CountSelect
        label="Layer count"
        value={draft.layers.length}
        onChange={(count) =>
          onChange({
            ...draft,
            layers: Array.from(
              { length: count },
              (_, index) => draft.layers[index] ?? { description: "" },
            ),
          })
        }
      />
      <div className="grid gap-3">
        {draft.layers.map((layer, index) => (
          <label key={index} className="grid gap-2 text-sm font-medium">
            Layer {index + 1} description
            <Textarea
              required
              className="resize-none"
              value={layer.description}
              onChange={(event) =>
                onChange({
                  ...draft,
                  layers: draft.layers.map((item, itemIndex) =>
                    itemIndex === index
                      ? { ...item, description: event.target.value }
                      : item,
                  ),
                })
              }
            />
          </label>
        ))}
      </div>
      <label className="grid gap-2 text-sm font-medium">
        Aspect ratio
        <Input
          required
          placeholder="e.g. 16:9"
          value={draft.aspectRatio}
          onChange={(event) =>
            onChange({ ...draft, aspectRatio: event.target.value })
          }
        />
      </label>
      <ImageDropzone
        fileName={draft.reference?.name}
        onSelect={(reference) => onChange({ ...draft, reference })}
        onClear={() => onChange({ ...draft, reference: undefined })}
      />
    </>
  );
}

function CountSelect({
  label,
  value,
  onChange,
}: {
  label: string;
  value: number;
  onChange: (count: number) => void;
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
            value={String(value)}
            onValueChange={(nextValue) => {
              onChange(Number(nextValue));
              setOpen(false);
            }}
          >
            {itemCounts.map((count) => (
              <DropdownMenuRadioItem key={count} value={String(count)}>
                {count}
              </DropdownMenuRadioItem>
            ))}
          </DropdownMenuRadioGroup>
        </DropdownMenuContent>
      </DropdownMenu>
    </label>
  );
}
