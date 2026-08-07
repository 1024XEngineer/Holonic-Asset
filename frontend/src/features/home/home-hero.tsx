const disciplines = [
  { label: "Characters", meta: "Sprites + motion" },
  { label: "Objects", meta: "Props + states" },
  { label: "Scenery", meta: "Scenes + layers" },
  { label: "Tilesets", meta: "Tiles + maps" },
  { label: "UI Set", meta: "HUD + menus" },
];

export function HomeHero() {
  return (
    <section className="relative overflow-hidden border-b bg-[#f0eee7] text-neutral-950">
      <div className="relative mx-auto grid min-h-[calc(100vh-3.5rem)] max-w-[100rem] grid-rows-[1fr_auto] px-5 sm:px-8 lg:px-10">
        <div className="grid items-center gap-10 py-14 lg:grid-cols-[minmax(0,.92fr)_minmax(30rem,.98fr)] lg:py-20">
          <div className="home-reveal">
            <h1 className="mt-0 max-w-none text-[clamp(2.4rem,3.2vw,4rem)] leading-[0.92] font-semibold tracking-[-0.045em] lg:whitespace-nowrap">
              Make the world Keep it yours.
            </h1>
            <div className="mt-9 max-w-3xl">
              <p className="max-w-xl text-base leading-7 text-neutral-600 sm:text-lg sm:leading-8">
                Holonic Asset helps you generate characters, objects,
                environments, tilesets, and UI Set assets—then keep every asset
                in a library with one consistent visual style.
              </p>
            </div>
            <div className="mt-8 flex flex-wrap items-center gap-3 font-mono text-[10px] font-semibold tracking-[0.13em] text-neutral-600">
              <span className="border border-neutral-950/20 px-3 py-2">
                PROJECT-BASED
              </span>
              <span className="border border-neutral-950/20 px-3 py-2">
                ENGINE-READY
              </span>
              <span className="flex items-center gap-2 px-2">
                <i className="size-2 rounded-full bg-lime-400" />
                CREATIVE SYSTEM ONLINE
              </span>
            </div>
          </div>

          <div className="home-reveal home-reveal-delay relative mx-auto w-full max-w-3xl lg:mx-0 lg:justify-self-end">
            <div className="relative aspect-[3/2] overflow-hidden rounded-[2rem] bg-neutral-950 p-2 shadow-[0_35px_80px_-40px_rgba(0,0,0,.75)]">
              <div className="relative size-full overflow-hidden rounded-[1.55rem]">
                <img
                  src="/project/reference/reference-exp.png"
                  alt="A complete game asset project preview"
                  className="absolute inset-0 size-full object-cover"
                />
                <div className="absolute inset-x-0 bottom-0 h-56 bg-gradient-to-t from-black/90 to-transparent" />
                <div className="absolute right-6 bottom-6 left-6 text-white">
                  <p className="font-mono text-[10px] tracking-[0.16em] text-lime-300">
                    COMPLETE GAME ASSET SYSTEM
                  </p>
                  <p className="mt-2 text-xl font-semibold">
                    One project. Every asset in sync.
                  </p>
                  <p className="mt-1 max-w-sm text-sm leading-6 text-white/65">
                    Generate characters, objects, scenery, tilesets, and UI Set
                    assets in one consistent style, then manage the complete set
                    together.
                  </p>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div className="grid border-t border-neutral-950/15 sm:grid-cols-2 lg:grid-cols-5">
          {disciplines.map(({ label, meta }, index) => (
            <div
              key={label}
              className="flex items-center gap-4 border-b border-neutral-950/15 py-5 last:border-b-0 sm:border-r sm:border-b-0 sm:px-6 sm:first:pl-0 sm:last:border-r-0"
            >
              <span className="font-mono text-[10px] text-neutral-400">
                0{index + 1}
              </span>
              <span>
                <span className="block text-sm font-semibold">{label}</span>
                <span className="mt-1 block font-mono text-[10px] uppercase tracking-[0.12em] text-neutral-500">
                  {meta}
                </span>
              </span>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
