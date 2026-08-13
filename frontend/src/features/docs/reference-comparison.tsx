import { useTranslation } from "react-i18next";

const results = [
  ["onlyPrompt", "without-reference-source.png", "onlyPromptDescription"],
  ["withReference", "with-reference-source.png", "withReferenceDescription"],
] as const;

export function ReferenceComparison() {
  const { t } = useTranslation("docs");
  return (
    <div className="mt-6 grid gap-4 sm:grid-cols-2">
      {results.map(([titleKey, image, descriptionKey]) => {
        const title = t(`reference.comparison.${titleKey}`);
        return (
          <figure
            key={image}
            className="flex flex-col border border-neutral-950/10 bg-[#f0eee7]"
          >
            <div className="flex items-baseline justify-between border-b border-neutral-950/10 bg-white px-4 py-3">
              <figcaption className="font-mono text-xs font-semibold tracking-[0.08em] text-neutral-950">
                {title}
              </figcaption>
            </div>
            <img
              src={`/project/reference/${image}`}
              alt={t("reference.comparison.imageAlt", { title })}
              loading="lazy"
              decoding="async"
              fetchPriority="low"
              className="block aspect-square w-full object-contain [image-rendering:pixelated]"
            />
            <p className="flex-1 border-t border-neutral-950/10 bg-white px-4 py-3 text-sm leading-6 text-neutral-600">
              {t(`reference.comparison.${descriptionKey}`)}
            </p>
          </figure>
        );
      })}
    </div>
  );
}
