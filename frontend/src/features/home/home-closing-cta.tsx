import { Link } from "@tanstack/react-router";
import { ArrowRight, Plus } from "lucide-react";

import { Button } from "@/components/ui/button";

export function HomeClosingCta() {
  return (
    <section
      aria-labelledby="closing-cta-heading"
      className="relative overflow-hidden border-b"
    >
      <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_center,color-mix(in_oklch,var(--foreground)_8%,transparent)_1px,transparent_1px)] bg-[size:22px_22px] [mask-image:radial-gradient(circle_at_center,black,transparent_68%)]" />
      <div className="relative mx-auto max-w-[100rem] px-5 py-24 text-center sm:px-8 sm:py-36 lg:px-10 lg:py-44">
        <p className="text-xs font-semibold tracking-[0.2em] text-muted-foreground">
          YOUR NEXT WORLD STARTS HERE
        </p>
        <h2
          id="closing-cta-heading"
          className="mx-auto mt-6 max-w-5xl text-5xl leading-[0.92] font-semibold tracking-[-0.065em] text-balance sm:text-7xl lg:text-8xl"
        >
          Make the first asset. Then build everything around it.
        </h2>
        <div className="mt-10 flex flex-wrap justify-center gap-3">
          <Button
            size="lg"
            render={
              <Link to="/projects/new" search={{ project: undefined, q: "" }} />
            }
          >
            <Plus data-icon="inline-start" />
            Start a project
          </Button>
          <Button size="lg" variant="outline" render={<Link to="/generate" />}>
            Try image generation
            <ArrowRight data-icon="inline-end" />
          </Button>
        </div>
      </div>
    </section>
  );
}
