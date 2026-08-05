const results = [
  {
    title: "only prompt (Without reference)",
    image: "without-reference-source.png",
    description:
      "Without a reference, the shared prompt alone does not reliably achieve the intended game's visual style or quality.",
  },
  {
    title: "prompt (With reference)",
    image: "with-reference-source.png",
    description:
      "Generated with the same prompt and a reference, producing a result more consistent with the game's visual style and quality.",
  },
] as const;

export function ReferenceComparison() {
  return (
    <div className="mt-6 grid gap-4 sm:grid-cols-2">
      {results.map(({ title, image, description }) => (
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
            alt={`${title} generated pixel-art character`}
            className="block aspect-square w-full object-contain [image-rendering:pixelated]"
          />
          <p className="flex-1 border-t border-neutral-950/10 bg-white px-4 py-3 text-sm leading-6 text-neutral-600">
            {description}
          </p>
        </figure>
      ))}
    </div>
  );
}
