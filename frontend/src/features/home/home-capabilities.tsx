import { useTranslation } from "react-i18next";
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
    key: "characters",
    title: "Characters",
    description:
      "Generate character prototypes (2/4/8-directional views), 4-16 frame animations (Idle, Walk, Attack), and bound action SFX.",
    detail: "Prototypes, Spritesheets & SFX",
    docsHref: "/docs/asset-concepts#character",
    icon: ImageIcon,
    preview: {
      src: "/assets/characters/basketballPlayer/running-4-frames_south.gif",
      alt: "Basketball player running south animation",
    },
  },
  {
    key: "objects",
    title: "Objects",
    description:
      "Generate distinctive game objects with consistent style, clear silhouettes, and the visual states needed for your world.",
    detail: "Game Objects & Visual States",
    docsHref: "/docs/asset-concepts#object",
    icon: Sparkles,
    preview: {
      src: "/assets/object/blue.gif",
      alt: "Animated blue object",
    },
  },
  {
    key: "scenery",
    title: "Scenery",
    description:
      "Generate multi-layer backgrounds with parallax sky, wind, and foreground detail for level design.",
    detail: "Parallax Scenery Layers",
    docsHref: "/docs/asset-concepts#scenery",
    icon: Layers3,
    preview: undefined,
  },
  {
    key: "tilesets",
    title: "Tilesets",
    description:
      "Build reusable pixel-art tilesets for streets, interiors, terrain, and seamless game-world construction.",
    detail: "Modular Tiles & Map Building",
    docsHref: "/docs/asset-concepts#tileset",
    icon: Layers3,
    preview: {
      src: "/assets/split_same_32px_grid_assets/Interior_Props_01.png",
      alt: "Pixel-art interior tileset preview",
    },
  },
  {
    key: "uiSet",
    title: "UI Set",
    description:
      "Design cohesive health bars, inventory panels, skill icon frames, system menus, and UI Set interaction sounds.",
    detail: "HUD, Controls & System Menus",
    docsHref: "/docs/asset-concepts#ui-set",
    icon: FolderKanban,
    preview: {
      src: "/assets/uiset/uiset.png",
      alt: "Pixel-art UI Set preview",
    },
  },
  {
    key: "audio",
    title: "Game Audio & BGM",
    description:
      "Generate action sound effects (SFX) matched with animations, plus immersive looping background music (BGM).",
    detail: "Animation SFX & Loopable BGM",
    docsHref: undefined,
    icon: Music,
    preview: undefined,
  },
] as const;

export function HomeCapabilities() {
  const { t } = useTranslation("home");
  return (
    <section aria-label={t("capabilitiesLabel")} className="bg-white">
      {capabilities.map(
        ({ docsHref, icon: Icon, key, preview, title }, index) => (
          <article
            key={title}
            className={`mx-auto flex min-h-[calc(100svh-3.5rem)] max-w-[110rem] flex-col border-x border-neutral-950/10 px-5 py-10 text-neutral-950 sm:px-10 sm:py-12 lg:px-16 ${index % 2 === 0 ? "bg-[#fcfbf7]" : "bg-[#edf4ef]"}`}
          >
            <div className="flex items-baseline justify-between border-b border-neutral-950/15 pb-5">
              <h2 className="text-4xl leading-none font-semibold tracking-[-0.045em] sm:text-5xl">
                {t(`capabilities.${key}.title`)}
              </h2>
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
                        alt={t("capabilities.scenery.alt")}
                        loading="lazy"
                        decoding="async"
                        fetchPriority="low"
                        className="absolute inset-0 size-full object-cover"
                      />
                      <img
                        src="/assets/wind-clean.png"
                        alt=""
                        loading="lazy"
                        decoding="async"
                        fetchPriority="low"
                        className="absolute inset-0 z-10 size-full origin-bottom scale-110 -translate-y-[38%] object-cover"
                      />
                      <img
                        src="/assets/nearby-trees-clean.png"
                        alt={t("capabilities.scenery.treesAlt")}
                        loading="lazy"
                        decoding="async"
                        fetchPriority="low"
                        className="absolute -left-[6%] -bottom-[10%] h-auto w-[112%] max-w-none mix-blend-multiply"
                      />
                    </div>
                  </div>
                ) : preview ? (
                  <img
                    src={preview.src}
                    alt={t(`capabilities.${key}.alt`)}
                    loading="lazy"
                    decoding="async"
                    fetchPriority="low"
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
                {docsHref ? (
                  <a
                    href={docsHref}
                    className="inline-flex items-center gap-2 font-mono text-xs tracking-[0.18em] text-cyan-700/70 transition-colors hover:text-cyan-800 hover:underline hover:underline-offset-4 focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-cyan-700"
                  >
                    {t(`capabilities.${key}.detail`)}
                    <ArrowUpRight className="size-3" aria-hidden="true" />
                  </a>
                ) : (
                  <p className="font-mono text-xs tracking-[0.18em] text-cyan-700/70">
                    {t(`capabilities.${key}.detail`)}
                  </p>
                )}
                <p className="mt-5 text-lg leading-8 text-slate-600 sm:text-xl sm:leading-9">
                  {t(`capabilities.${key}.description`)}
                </p>
              </div>
            </div>
          </article>
        ),
      )}
    </section>
  );
}
