import type { ReactNode } from "react";

export function ProjectStartCard({
  description,
  icon,
  onSelect,
  title,
}: {
  description: string;
  icon: ReactNode;
  onSelect: () => void;
  title: string;
}) {
  return (
    <button
      type="button"
      className="flex min-h-52 flex-col rounded-md border bg-card p-6 text-left shadow-sm transition-[transform,border-color,box-shadow] hover:-translate-y-1 hover:border-foreground hover:shadow-xl focus-visible:outline-3 focus-visible:outline-ring focus-visible:outline-offset-3"
      onClick={onSelect}
    >
      <span className="grid size-10 place-items-center rounded-md bg-muted text-foreground">
        {icon}
      </span>
      <h2 className="mt-auto text-base font-semibold">{title}</h2>
      <p className="mt-2 text-sm leading-6 text-muted-foreground">
        {description}
      </p>
    </button>
  );
}
