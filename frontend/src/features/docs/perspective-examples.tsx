type PerspectiveExampleProps = {
  image: "top-down-2d.jpg" | "side-on.jpg" | "isometric.png";
  alt: string;
};

export function PerspectiveExample({ image, alt }: PerspectiveExampleProps) {
  return (
    <figure className="mt-6 border border-neutral-950/10 bg-[#f0eee7]">
      <img
        src={`/project/perspective/${image}`}
        alt={alt}
        className="block h-auto w-full"
      />
    </figure>
  );
}
