import { Link } from "@tanstack/react-router";
import {
  ArrowUpRight,
  FolderKanban,
  ImageIcon,
  Layers3,
  Music,
  Sparkles,
} from "lucide-react";

const capabilities = [
  {
    title: "Characters",
    description:
      "Generate character prototypes (1/4/8-directional views), 4-16 frame animations (Idle, Walk, Attack), and bound action SFX.",
    detail: "Prototypes, Spritesheets & SFX",
    to: "/projects",
    icon: ImageIcon,
    preview: {
      src: "/assets/characters/basketballPlayer/running-4-frames_south.gif",
      alt: "Basketball player running south animation",
    },
  },
  {
    title: "Objects",
    description:
      "Generate distinctive game objects with consistent style, clear silhouettes, and the visual states needed for your world.",
    detail: "Game Objects & Visual States",
    to: "/projects",
    icon: Sparkles,
    preview: {
      src: "/assets/object/blue.gif",
      alt: "Animated blue object",
    },
  },
  {
    title: "Scenery",
    description:
      "Generate multi-layer backgrounds with parallax sky, wind, and foreground detail for level design.",
    detail: "Parallax Scenery Layers",
    to: "/projects",
    icon: Layers3,
    preview: undefined,
  },
  {
    title: "Tilesets",
    description:
      "Build reusable pixel-art tilesets for streets, interiors, terrain, and seamless game-world construction.",
    detail: "Modular Tiles & Map Building",
    to: "/projects",
    icon: Layers3,
    preview: {
      src: "/assets/split_same_32px_grid_assets/Interior_Props_01.png",
      alt: "Pixel-art interior tileset preview",
    },
  },
  {
    title: "Game UI",
    description:
      "Design cohesive health bars, inventory panels, skill icon frames, system menus, and UI interaction sounds.",
    detail: "HUD, Controls & System Menus",
    to: "/projects",
    icon: FolderKanban,
    preview: {
      src: "/assets/ui/ui.png",
      alt: "Pixel-art game UI preview",
    },
  },
  {
    title: "Game Audio & BGM",
    description:
      "Generate action sound effects (SFX) matched with animations, plus immersive looping background music (BGM).",
    detail: "Animation SFX & Loopable BGM",
    to: "/projects",
    icon: Music,
    preview: undefined,
  },
] as const;

export function HomeCapabilities() {
  return (
    <section aria-label="Asset creation capabilities" className="bg-white">
      {capabilities.map(
        ({ description, detail, icon: Icon, preview, title, to }, index) => (
          <article
            key={title}
            className={`mx-auto flex min-h-[calc(100svh-3.5rem)] max-w-[110rem] flex-col border-x border-neutral-950/10 px-5 py-10 text-neutral-950 sm:px-10 sm:py-12 lg:px-16 ${index % 2 === 0 ? "bg-[#fcfbf7]" : "bg-[#edf4ef]"}`}
          >
            <div className="flex items-baseline justify-between border-b border-neutral-950/15 pb-5">
              <div className="text-4xl leading-none font-semibold tracking-[-0.045em] sm:text-5xl">
                {title}
              </div>
              <span className="font-mono text-xs text-neutral-400">
                0{index + 1}
              </span>
            </div>

            <div className="grid flex-1 items-center gap-14 py-12 lg:grid-cols-[minmax(18rem,.8fr)_minmax(28rem,1fr)] lg:gap-24 lg:py-20">
              <div className="flex min-h-56 items-center justify-center lg:min-h-80">
                {title === "Scenery" ? (
                  <div className="w-[92%]">
                    <div className="relative aspect-[16/10] overflow-hidden rounded-xl">
                      <img
                        src="/assets/sky.png"
                        alt="Pixel-art scenery sky"
                        className="absolute inset-0 size-full object-cover"
                      />
                      <img
                        src="/assets/wind-clean.png"
                        alt=""
                        className="absolute inset-0 z-10 size-full origin-bottom scale-110 -translate-y-[38%] object-cover"
                      />
                      <img
                        src="/assets/nearby-trees-clean.png"
                        alt="Pixel-art scenery trees"
                        className="absolute -left-[6%] -bottom-[10%] h-auto w-[112%] max-w-none mix-blend-multiply"
                      />
                    </div>
                  </div>
                ) : preview ? (
                  <img
                    src={preview.src}
                    alt={preview.alt}
                    className={`aspect-square object-contain [image-rendering:pixelated] ${title === "Objects" ? "w-[62%] sm:w-[68%]" : "w-[76%] sm:w-[82%]"}`}
                  />
                ) : (
                  <Icon
                    aria-hidden="true"
                    className="size-28 text-cyan-700 sm:size-36"
                    strokeWidth={1.25}
                  />
                )}
              </div>

              <div className="max-w-2xl">
                <p className="font-mono text-xs tracking-[0.18em] text-cyan-700/70">
                  {detail}
                </p>
                <h2 className="mt-5 text-3xl leading-tight font-semibold tracking-[-0.035em] sm:text-4xl">
                  {index === 0 ? "Make your characters move" : title}
                </h2>
                <p className="mt-6 text-lg leading-8 text-slate-600 sm:text-xl sm:leading-9">
                  {description}
                </p>
                <Link
                  to={to}
                  search={{ project: undefined, q: "" }}
                  className="mt-7 inline-flex h-12 items-center gap-2 rounded-lg border border-cyan-700 px-5 text-base font-semibold text-cyan-700 transition-colors hover:bg-cyan-700 hover:text-white focus-visible:ring-3 focus-visible:ring-cyan-700/50 focus-visible:outline-none"
                >
                  Explore assets
                  <ArrowUpRight className="size-4" />
                </Link>
              </div>
            </div>
          </article>
        ),
      )}
    </section>
  );
}
