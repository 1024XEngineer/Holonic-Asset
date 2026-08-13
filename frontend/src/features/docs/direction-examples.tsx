import { useTranslation } from "react-i18next";

const fourDirections = [
  ["front", "idle-front.png"],
  ["right", "idle-right.png"],
  ["back", "idle-back.png"],
  ["left", "idle-left.png"],
] as const;

const twoDirections = [
  ["left", "idle-left.png"],
  ["right", "idle-right.png"],
] as const;

const eightDirections = [
  "north-west",
  "north",
  "north-east",
  "west",
  "east",
  "south-west",
  "south",
  "south-east",
] as const;

export function TwoDirectionExample({
  priority = false,
}: {
  priority?: boolean;
}) {
  const { t } = useTranslation("docs");
  return (
    <div className="mt-6 grid grid-cols-2 gap-3 sm:grid-cols-4">
      {twoDirections.map(([direction, fileName], index) => (
        <div
          key={direction}
          className="relative grid aspect-square place-items-start overflow-hidden border border-neutral-950/10 bg-[#f0eee7]"
        >
          <img
            src={`/assets/characters/swordsman/${fileName}`}
            alt={t("directions.labels.swordsmanAlt", {
              label: t(`directions.labels.${direction}`),
            })}
            loading={priority && index === 0 ? "eager" : "lazy"}
            decoding="async"
            fetchPriority={priority && index === 0 ? "high" : "low"}
            className="h-full max-w-none w-auto [image-rendering:pixelated]"
          />
          <span className="absolute right-2 bottom-2 border border-neutral-950/15 bg-white/85 px-2 py-1 font-mono text-[10px] font-semibold tracking-[0.1em] text-neutral-700">
            {t(`directions.labels.${direction}`)}
          </span>
        </div>
      ))}
    </div>
  );
}

export function FourDirectionExample() {
  const { t } = useTranslation("docs");
  return (
    <div className="mt-6 grid grid-cols-2 gap-3 sm:grid-cols-4">
      {fourDirections.map(([direction, fileName]) => (
        <div
          key={direction}
          className="relative grid aspect-square place-items-start overflow-hidden border border-neutral-950/10 bg-[#f0eee7]"
        >
          <img
            src={`/assets/characters/swordsman/${fileName}`}
            alt={t("directions.labels.swordsmanAlt", {
              label: t(`directions.labels.${direction}`),
            })}
            loading="lazy"
            decoding="async"
            fetchPriority="low"
            className="h-full max-w-none w-auto [image-rendering:pixelated]"
          />
          <span className="absolute right-2 bottom-2 border border-neutral-950/15 bg-white/85 px-2 py-1 font-mono text-[10px] font-semibold tracking-[0.1em] text-neutral-700">
            {t(`directions.labels.${direction}`)}
          </span>
        </div>
      ))}
    </div>
  );
}

export function EightDirectionExample() {
  const { t } = useTranslation("docs");
  return (
    <div className="mt-6 grid grid-cols-2 gap-3 sm:grid-cols-4">
      {eightDirections.map((direction) => (
        <div
          key={direction}
          className="relative grid aspect-square place-items-center border border-neutral-950/10 bg-[#f0eee7] p-3"
        >
          <img
            src={`/assets/characters/basketballPlayer/rotations/${direction}.png`}
            alt={t("directions.labels.basketballAlt", {
              label: t(`directions.labels.${direction}`),
            })}
            loading="lazy"
            decoding="async"
            fetchPriority="low"
            className="size-full object-contain [image-rendering:pixelated]"
          />
          <span className="absolute right-2 bottom-2 border border-neutral-950/15 bg-white/85 px-2 py-1 font-mono text-[10px] font-semibold tracking-[0.1em] text-neutral-700">
            {t(`directions.labels.${direction}`)}
          </span>
        </div>
      ))}
    </div>
  );
}
