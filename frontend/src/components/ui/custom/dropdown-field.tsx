/**
 * Custom Component: DropdownField
 * Reusable single-select dropdown menu component built on top of shadcn/ui DropdownMenu.
 */

import { ChevronDown } from "lucide-react";
import { useEffect, useState, type ReactNode } from "react";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { cn } from "@/lib/utils";

export function DropdownField({
  label,
  value,
  options,
  onChange,
  className,
  disabled = false,
  size = "default",
}: {
  label: ReactNode;
  value: string;
  options: readonly string[];
  onChange: (value: string) => void;
  className?: string;
  disabled?: boolean;
  size?: "default" | "compact";
}) {
  const [open, setOpen] = useState(false);

  useEffect(() => {
    if (disabled) setOpen(false);
  }, [disabled]);

  return (
    <label
      className={cn(
        "grid gap-2 font-medium",
        size === "compact" ? "text-xs text-muted-foreground" : "text-sm",
        className,
      )}
    >
      <span className={cn(size === "compact" && "flex items-center gap-1.5")}>
        {label}
      </span>
      <DropdownMenu open={open} onOpenChange={setOpen}>
        <DropdownMenuTrigger
          render={
            <Button
              disabled={disabled}
              type="button"
              variant="outline"
              className={cn(
                "w-full justify-between font-normal",
                size === "compact" ? "h-8 px-2.5" : "h-9 px-3",
              )}
            />
          }
        >
          <span className="truncate">{value || "Not specified"}</span>
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
            {options.map((option) => (
              <DropdownMenuRadioItem key={option} value={option}>
                {option}
              </DropdownMenuRadioItem>
            ))}
          </DropdownMenuRadioGroup>
        </DropdownMenuContent>
      </DropdownMenu>
    </label>
  );
}
