const fourDirections = [
  ["front", "idle-front.png", "Front"],
  ["right", "idle-right.png", "Right"],
  ["back", "idle-back.png", "Back"],
  ["left", "idle-left.png", "Left"],
] as const;

const eightDirections = [
  ["north-west", "North-west"],
  ["north", "North"],
  ["north-east", "North-east"],
  ["west", "West"],
  ["east", "East"],
  ["south-west", "South-west"],
  ["south", "South"],
  ["south-east", "South-east"],
] as const;

export function FourDirectionExample() {
  return (
    <div className="mt-6 grid grid-cols-2 gap-3 sm:grid-cols-4">
      {fourDirections.map(([direction, fileName, label]) => (
        <div
          key={direction}
          className="relative grid aspect-square place-items-start overflow-hidden border border-neutral-950/10 bg-[#f0eee7]"
        >
          <img
            src={`/assets/characters/swordsman/${fileName}`}
            alt={`${label} swordsman direction`}
            className="h-full max-w-none w-auto [image-rendering:pixelated]"
          />
          <span className="absolute right-2 bottom-2 border border-neutral-950/15 bg-white/85 px-2 py-1 font-mono text-[10px] font-semibold tracking-[0.1em] text-neutral-700">
            {label}
          </span>
        </div>
      ))}
    </div>
  );
}

export function EightDirectionExample() {
  return (
    <div className="mt-6 grid grid-cols-2 gap-3 sm:grid-cols-4">
      {eightDirections.map(([direction, label]) => (
        <div
          key={direction}
          className="relative grid aspect-square place-items-center border border-neutral-950/10 bg-[#f0eee7] p-3"
        >
          <img
            src={`/assets/characters/basketballPlayer/rotations/${direction}.png`}
            alt={`${label} basketball player direction`}
            className="size-full object-contain [image-rendering:pixelated]"
          />
          <span className="absolute right-2 bottom-2 border border-neutral-950/15 bg-white/85 px-2 py-1 font-mono text-[10px] font-semibold tracking-[0.1em] text-neutral-700">
            {label}
          </span>
        </div>
      ))}
    </div>
  );
}
