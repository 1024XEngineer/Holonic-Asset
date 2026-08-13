import { useTranslation } from "react-i18next";

export function HomeHero() {
  const { t } = useTranslation("home");
  const disciplines = [
    "characters",
    "objects",
    "scenery",
    "tilesets",
    "uiSet",
  ] as const;

  return (
    <section className="relative overflow-hidden border-b bg-[#f0eee7] text-neutral-950">
      <div className="relative mx-auto grid min-h-[calc(100vh-3.5rem)] max-w-[100rem] grid-rows-[1fr_auto] px-5 sm:px-8 lg:px-10">
        <div className="grid items-center gap-10 py-14 lg:grid-cols-[minmax(0,.92fr)_minmax(30rem,.98fr)] lg:py-20">
          <div className="home-reveal">
            <h1 className="mt-0 max-w-none text-[clamp(2.4rem,3.2vw,4rem)] leading-[0.92] font-semibold tracking-[-0.045em] lg:whitespace-nowrap">
              {t("heroTitle")}
            </h1>
            <div className="mt-9 max-w-3xl">
              <p className="max-w-xl text-base leading-7 text-neutral-600 sm:text-lg sm:leading-8">
                {t("heroDescription")}
              </p>
            </div>
            <div className="mt-8 flex flex-wrap items-center gap-3 font-mono text-[10px] font-semibold tracking-[0.13em] text-neutral-600">
              <span className="border border-neutral-950/20 px-3 py-2">
                {t("projectBased")}
              </span>
              <span className="border border-neutral-950/20 px-3 py-2">
                {t("engineReady")}
              </span>
              <span className="flex items-center gap-2 px-2">
                <i className="size-2 rounded-full bg-lime-400" />
                {t("systemOnline")}
              </span>
            </div>
          </div>

          <div className="home-reveal home-reveal-delay relative mx-auto w-full max-w-3xl lg:mx-0 lg:justify-self-end">
            <div className="relative aspect-[3/2] overflow-hidden rounded-[2rem] bg-neutral-950 p-2 shadow-[0_35px_80px_-40px_rgba(0,0,0,.75)]">
              <div className="relative size-full overflow-hidden rounded-[1.55rem]">
                <img
                  src="/project/reference/reference-exp.png"
                  alt={t("previewAlt")}
                  loading="eager"
                  decoding="async"
                  fetchPriority="high"
                  className="absolute inset-0 size-full object-cover"
                />
                <div className="absolute inset-x-0 bottom-0 h-56 bg-gradient-to-t from-black/90 to-transparent" />
                <div className="absolute right-6 bottom-6 left-6 text-white">
                  <p className="font-mono text-[10px] tracking-[0.16em] text-lime-300">
                    {t("completeSystem")}
                  </p>
                  <p className="mt-2 text-xl font-semibold">{t("syncTitle")}</p>
                  <p className="mt-1 max-w-sm text-sm leading-6 text-white/65">
                    {t("syncDescription")}
                  </p>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div className="grid border-t border-neutral-950/15 sm:grid-cols-2 lg:grid-cols-5">
          {disciplines.map((discipline, index) => (
            <div
              key={discipline}
              className="flex items-center gap-4 border-b border-neutral-950/15 py-5 last:border-b-0 sm:border-r sm:border-b-0 sm:px-6 sm:first:pl-0 sm:last:border-r-0"
            >
              <span className="font-mono text-[10px] text-neutral-400">
                0{index + 1}
              </span>
              <span>
                <span className="block text-sm font-semibold">
                  {t(`disciplines.${discipline}.label`)}
                </span>
                <span className="mt-1 block font-mono text-[10px] uppercase tracking-[0.12em] text-neutral-500">
                  {t(`disciplines.${discipline}.detail`)}
                </span>
              </span>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
