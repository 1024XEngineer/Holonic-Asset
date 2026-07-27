const disciplines = [
  "Characters, Animations & SFX",
  "Scenery & Seamless Tilesets",
  "Game UI & HUD Components",
  "Project Visual Direction Context",
];

export function HomeHero() {
  return (
    <section className="relative overflow-hidden border-b bg-[#f0eee7] text-neutral-950">
      <div className="pointer-events-none absolute inset-0 bg-[linear-gradient(to_right,transparent_calc(100%-1px),rgba(0,0,0,.07)_1px)] bg-[size:12.5%_100%]" />

      <div className="relative mx-auto grid min-h-[calc(100vh-3.5rem)] max-w-[100rem] grid-rows-[1fr_auto] px-5 sm:px-8 lg:px-10">
        <div className="grid items-center gap-10 py-14 lg:grid-cols-[minmax(0,1.04fr)_minmax(28rem,.72fr)] lg:py-20">
          <div>
            <p className="flex items-center gap-3 text-xs font-semibold tracking-[0.2em]">
              <img
                src="/logos/logo-light-nobackground.svg"
                alt=""
                className="size-7 shrink-0"
              />
              <span className="size-2 bg-lime-400" />
              HOLONIC ASSET
            </p>
            <h1 className="mt-8 max-w-5xl text-[clamp(3.8rem,8.5vw,8.5rem)] leading-[0.82] font-semibold tracking-[-0.075em] text-balance">
              Make the world.
              <br />
              Keep it yours.
            </h1>
            <div className="mt-9 max-w-3xl">
              <p className="max-w-xl text-base leading-7 text-neutral-600 sm:text-lg sm:leading-8">
                Holonic Asset helps you generate characters, objects,
                environments, tilesets, and game UI—then keep every asset in a
                library with a clear visual direction.
              </p>
            </div>
          </div>

          <div className="relative mx-auto w-full max-w-xl lg:mx-0 lg:justify-self-end">
            <div className="absolute -top-5 -left-5 z-10 hidden border border-neutral-950 bg-lime-400 px-4 py-2 font-mono text-[10px] font-semibold tracking-[0.14em] sm:block">
              ASSET COLLECTION / READY
            </div>
            <div className="relative aspect-[4/5] overflow-hidden rounded-[2rem] bg-neutral-950 p-2 shadow-[0_35px_80px_-40px_rgba(0,0,0,.75)]">
              <div className="relative size-full overflow-hidden rounded-[1.55rem]">
                <img
                  src="/assets/sky.png"
                  alt=""
                  className="absolute inset-0 size-full object-cover"
                />
                <img
                  src="/assets/wind.png"
                  alt=""
                  className="absolute inset-0 size-full object-cover mix-blend-multiply"
                />
                <img
                  src="/assets/nearby-trees.png"
                  alt="Layered game environment created in Holonic Asset"
                  className="absolute inset-0 size-full object-cover mix-blend-multiply"
                />
                <div className="absolute inset-x-0 bottom-0 h-48 bg-gradient-to-t from-black/80 to-transparent" />
                <div className="absolute right-6 bottom-6 left-6 text-white">
                  <p className="font-mono text-[10px] tracking-[0.16em] text-white/55">
                    SCENERY / EDITABLE LAYERS
                  </p>
                  <p className="mt-2 text-xl font-semibold">
                    Build scenes in layers
                  </p>
                  <p className="mt-1 text-sm text-white/60">
                    Generate a background, then refine each visual layer.
                  </p>
                </div>
              </div>
            </div>
            <div className="absolute -right-4 -bottom-4 grid size-24 place-items-center rounded-full border border-neutral-950 bg-[#f0eee7] text-center font-mono text-[10px] leading-4 font-semibold tracking-[0.12em]">
              BUILT FOR
              <br />
              GAME MAKERS
            </div>
          </div>
        </div>

        <div className="grid border-t border-neutral-950/15 sm:grid-cols-2 lg:grid-cols-4">
          {disciplines.map((discipline, index) => (
            <div
              key={discipline}
              className="flex items-center gap-4 border-b border-neutral-950/15 py-5 last:border-b-0 sm:border-r sm:border-b-0 sm:px-6 sm:first:pl-0 sm:last:border-r-0"
            >
              <span className="font-mono text-[10px] text-neutral-400">
                0{index + 1}
              </span>
              <span className="text-sm font-semibold">{discipline}</span>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
