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
    number: "01",
    title: "Characters & Animations",
    description:
      "Generate character prototypes (1/4/8-directional views), 4-16 frame animations (Idle, Walk, Attack), and bound action SFX.",
    detail: "Prototypes, Spritesheets & SFX",
    to: "/projects",
    icon: ImageIcon,
  },
  {
    number: "02",
    title: "Objects & Props",
    description:
      "Create interactive props, chests, equipment, and items with custom state changes and item interaction audio.",
    detail: "Items, Chests & Interactive Props",
    to: "/projects",
    icon: Sparkles,
  },
  {
    number: "03",
    title: "Scenery & Tilesets",
    description:
      "Generate multi-layer backgrounds (parallax scrolling sky, mountains, trees) and seamless map tiles for level design.",
    detail: "Parallax Layers & Seamless Maps",
    to: "/projects",
    icon: Layers3,
  },
  {
    number: "04",
    title: "Game UI & HUD",
    description:
      "Design cohesive health bars, inventory panels, skill icon frames, system menus, and UI interaction sounds.",
    detail: "HUD, Controls & System Menus",
    to: "/projects",
    icon: FolderKanban,
  },
  {
    number: "05",
    title: "Game Audio & BGM",
    description:
      "Generate action sound effects (SFX) matched with animations, plus immersive looping background music (BGM).",
    detail: "Animation SFX & Loopable BGM",
    to: "/projects",
    icon: Music,
  },
] as const;

export function HomeCapabilities() {
  return (
    <section
      aria-labelledby="capabilities-heading"
      className="mx-auto max-w-[100rem] px-5 py-20 sm:px-8 sm:py-28 lg:px-10 lg:py-36"
    >
      <div className="grid gap-10 lg:grid-cols-[minmax(0,.7fr)_minmax(0,1.3fr)]">
        <div>
          <p className="text-xs font-semibold tracking-[0.2em] text-muted-foreground">
            ASSET CREATION PIPELINE
          </p>
          <h2
            id="capabilities-heading"
            className="mt-5 max-w-lg text-4xl leading-[1.02] font-semibold tracking-[-0.045em] sm:text-5xl"
          >
            Complete 2D Game Asset Generation
          </h2>
        </div>
        <p className="max-w-2xl self-end text-base leading-7 text-muted-foreground sm:text-lg sm:leading-8">
          From multi-directional character prototypes to layered scenery, tile
          maps, game UI, and bound audio—every asset is built to fit your
          project&apos;s unified visual direction.
        </p>
      </div>

      <div className="mt-14 border-t">
        {capabilities.map(
          ({ description, detail, icon: Icon, number, title, to }) => (
            <Link
              key={number}
              to={to}
              search={{ project: undefined, q: "" }}
              className="group grid gap-6 border-b py-8 transition-colors hover:bg-muted/40 focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none sm:grid-cols-[4rem_minmax(12rem,.6fr)_minmax(0,1fr)_auto] sm:items-center sm:px-4 lg:py-11"
            >
              <span className="font-mono text-xs text-muted-foreground">
                {number}
              </span>
              <div className="flex items-center gap-4">
                <span className="grid size-10 place-items-center rounded-full border bg-background">
                  <Icon className="size-4" />
                </span>
                <h3 className="text-xl font-semibold tracking-tight">
                  {title}
                </h3>
              </div>
              <p className="max-w-xl text-sm leading-6 text-muted-foreground">
                {description}
              </p>
              <div className="flex items-center justify-between gap-5 sm:justify-end">
                <span className="text-xs font-medium text-muted-foreground">
                  {detail}
                </span>
                <ArrowUpRight className="size-4 transition-transform group-hover:translate-x-1 group-hover:-translate-y-1" />
              </div>
            </Link>
          ),
        )}
      </div>
    </section>
  );
}
