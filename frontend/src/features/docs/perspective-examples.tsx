import { useTranslation } from "react-i18next";
import type { DocsKey } from "./doc-text";

type PerspectiveExampleProps = {
  image: "top-down-2d.jpg" | "side-on.jpg" | "isometric.png";
  altKey: DocsKey;
  priority?: boolean;
};

const imageDimensions = {
  "top-down-2d.jpg": { width: 1800, height: 1200 },
  "side-on.jpg": { width: 1800, height: 1200 },
  "isometric.png": { width: 1403, height: 814 },
} as const;

export function PerspectiveExample({
  image,
  altKey,
  priority = false,
}: PerspectiveExampleProps) {
  const { t } = useTranslation("docs");
  const { width, height } = imageDimensions[image];
  return (
    <figure className="mt-6 border border-neutral-950/10 bg-[#f0eee7]">
      <img
        src={`/project/perspective/${image}`}
        alt={t(altKey)}
        width={width}
        height={height}
        loading={priority ? "eager" : "lazy"}
        decoding="async"
        fetchPriority={priority ? "high" : "low"}
        className="block h-auto w-full"
      />
    </figure>
  );
}
