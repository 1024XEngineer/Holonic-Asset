import { Link } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";

export function LoginArtwork() {
  const { t } = useTranslation("auth");

  return (
    <section className="relative min-h-64 overflow-hidden border-b border-black/15 lg:min-h-screen lg:border-r lg:border-b-0 dark:border-white/15">
      <img
        src="/project/reference.png"
        alt={t("artworkAlt")}
        className="absolute inset-0 size-full object-cover [image-rendering:auto]"
      />
      <div className="absolute inset-0 bg-black/15" />
      <div className="absolute inset-x-0 bottom-0 h-1/2 bg-[linear-gradient(to_top,rgba(11,17,10,0.82),transparent)]" />
      <Link
        to="/"
        aria-label={t("homeAriaLabel")}
        className="absolute top-5 left-5 inline-flex rounded-md bg-white/92 px-3 py-2 shadow-sm ring-1 ring-black/10 backdrop-blur focus-visible:ring-3 focus-visible:ring-white/70 focus-visible:outline-none sm:top-8 sm:left-8"
      >
        <img
          src="/logos/logo-light-with-name.svg"
          alt=""
          className="h-5 w-auto"
        />
      </Link>
      <div className="absolute right-5 bottom-5 left-5 text-white sm:right-8 sm:bottom-8 sm:left-8 lg:right-12 lg:bottom-12 lg:left-12">
        <p className="text-xs font-semibold tracking-[0.18em] uppercase text-white/75">
          {t("artworkEyebrow")}
        </p>
        <p className="mt-2 max-w-xl text-lg leading-7 font-semibold sm:text-xl">
          {t("artworkCaption")}
        </p>
      </div>
    </section>
  );
}
