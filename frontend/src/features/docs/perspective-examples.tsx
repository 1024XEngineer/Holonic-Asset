import { useTranslation } from "react-i18next";

type PerspectiveExampleProps = {
  image: "top-down-2d.jpg" | "side-on.jpg" | "isometric.png";
  altKey: string;
};

export function PerspectiveExample({ image, altKey }: PerspectiveExampleProps) {
  const { t } = useTranslation("docs");
  return (
    <figure className="mt-6 border border-neutral-950/10 bg-[#f0eee7]">
      <img
        src={`/project/perspective/${image}`}
        alt={t(altKey)}
        className="block h-auto w-full"
      />
    </figure>
  );
}
