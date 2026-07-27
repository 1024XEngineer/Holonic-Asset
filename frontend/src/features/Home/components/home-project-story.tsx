import { Link } from "@tanstack/react-router";
import { ArrowRight, Layers3 } from "lucide-react";

import { Button } from "@/components/ui/button";

const layers = [
  { label: "Foreground", color: "bg-emerald-300" },
  { label: "Atmosphere", color: "bg-amber-200" },
  { label: "Background", color: "bg-sky-300" },
];

export function HomeProjectStory() {
  return (
    <section
      aria-labelledby="project-story-heading"
      className="overflow-hidden bg-neutral-950 text-white"
    >
      <div className="mx-auto grid max-w-[100rem] lg:grid-cols-2">
        <div className="relative min-h-[34rem] overflow-hidden border-b border-white/15 lg:min-h-[48rem] lg:border-r lg:border-b-0">
          <img
            src="/assets/sky.png"
            alt=""
            className="absolute inset-0 size-full object-cover"
          />
          <img
            src="/assets/wind.png"
            alt=""
            className="absolute inset-0 size-full object-cover mix-blend-multiply"
          />
          <img
            src="/assets/nearby-trees.png"
            alt="A multi-layer game scenery project"
            className="absolute inset-0 size-full object-cover mix-blend-multiply"
          />
          <div className="absolute inset-0 bg-gradient-to-t from-black/85 via-transparent to-black/20" />

          <div className="absolute top-5 right-5 left-5 flex items-center justify-between border-b border-white/25 pb-4 font-mono text-[10px] tracking-[0.14em] text-white/65">
            <span>ASSET LIBRARY / SCENERY</span>
            <span>ORGANIZED BY PROJECT</span>
          </div>

          <div className="absolute right-5 bottom-5 left-5 sm:right-8 sm:bottom-8 sm:left-8">
            <div className="max-w-sm rounded-2xl border border-white/15 bg-black/55 p-5 backdrop-blur-md">
              <div className="flex items-center gap-2">
                <Layers3 className="size-4 text-white/60" />
                <p className="text-xs font-semibold tracking-[0.14em] text-white/60">
                  SCENE LAYERS
                </p>
              </div>
              <div className="mt-4 grid gap-2">
                {layers.map(({ color, label }) => (
                  <div
                    key={label}
                    className="flex items-center gap-3 border-t border-white/10 pt-2 text-sm"
                  >
                    <span className={`size-2 rounded-full ${color}`} />
                    <span>{label}</span>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>

        <div className="flex flex-col justify-between px-5 py-14 sm:px-8 sm:py-20 lg:px-14 lg:py-24">
          <div>
            <p className="text-xs font-semibold tracking-[0.2em] text-lime-300">
              CONSISTENCY IS A FEATURE
            </p>
            <h2
              id="project-story-heading"
              className="mt-6 max-w-xl text-4xl leading-[1.02] font-semibold tracking-[-0.05em] sm:text-5xl lg:text-6xl"
            >
              Not a folder of outputs. An asset library with direction.
            </h2>
          </div>

          <div className="mt-20 max-w-xl lg:mt-28">
            <p className="text-lg leading-8 text-white/70">
              Every project records the game type, platform, visual style, and
              visual direction that guide new generations. Assets stay grouped
              by type, from characters and objects to scenery, tiles, and UI.
            </p>
            <p className="mt-5 text-sm leading-7 text-white/50">
              Keep useful details attached to every asset: canvas size,
              perspective, tags, scene layers, and revision history. Your
              library stays understandable as the game grows.
            </p>
            <Button
              className="mt-8 bg-white text-neutral-950 hover:bg-lime-300"
              render={
                <Link to="/projects" search={{ project: undefined, q: "" }} />
              }
            >
              See the project workspace
              <ArrowRight data-icon="inline-end" />
            </Button>
          </div>
        </div>
      </div>
    </section>
  );
}
