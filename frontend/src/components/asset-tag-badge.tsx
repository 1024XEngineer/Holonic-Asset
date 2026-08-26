import { X } from "lucide-react";
import type { ReactNode } from "react";

import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import { getTagBadgeStyle, type AssetTag } from "@/model/asset";

export function AssetTagBadge({
  children,
  className,
  disabled = false,
  onRemove,
  tag,
  variant = "github",
}: {
  children?: ReactNode;
  className?: string;
  disabled?: boolean;
  onRemove?: () => void;
  tag: AssetTag;
  variant?: "github" | "outline";
}) {
  const badgeStyle =
    variant === "github" ? getTagBadgeStyle(tag.color) : undefined;

  return (
    <Badge
      variant="outline"
      style={badgeStyle}
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-xs font-medium text-foreground transition-colors",
        onRemove && "pr-1",
        className,
      )}
      title={tag.description || tag.name}
    >
      <span
        className="size-1.5 shrink-0 rounded-full"
        style={{ backgroundColor: tag.color }}
        aria-hidden="true"
      />
      <span className="truncate">{tag.name}</span>
      {children}
      {onRemove ? (
        <button
          type="button"
          disabled={disabled}
          className="grid size-3.5 place-items-center rounded-full text-muted-foreground/80 transition-colors hover:bg-black/10 hover:text-foreground dark:hover:bg-white/20 disabled:pointer-events-none"
          aria-label={`Remove ${tag.name}`}
          onClick={(e) => {
            e.stopPropagation();
            onRemove();
          }}
        >
          <X className="size-2.5" aria-hidden="true" />
        </button>
      ) : null}
    </Badge>
  );
}
