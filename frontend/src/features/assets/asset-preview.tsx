import { cn } from "@/lib/utils";

export function AssetPreview({
  accentClassName,
  className,
}: {
  accentClassName: string;
  className?: string;
}) {
  return (
    <div
      className={cn(
        "grid aspect-[16/10] place-items-center bg-[linear-gradient(135deg,#ffffff_0%,#efe7d8_48%,#d8e7df_100%)]",
        className,
      )}
    >
      <div className="grid size-16 grid-cols-4 grid-rows-4 gap-1 rounded-md border bg-white/70 p-2">
        {Array.from({ length: 16 }).map((_, index) => (
          <span
            key={index}
            className={cn(
              "rounded-[2px]",
              index % 3 === 0
                ? accentClassName
                : index % 2 === 0
                  ? "bg-muted"
                  : "bg-card",
            )}
          />
        ))}
      </div>
    </div>
  );
}
