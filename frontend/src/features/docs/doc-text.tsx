import type { ParseKeys } from "i18next";
import { useTranslation } from "react-i18next";

export type DocsKey = ParseKeys<"docs">;

export function DocText({ i18nKey }: { i18nKey: DocsKey }) {
  const { t } = useTranslation("docs");
  return <>{t(i18nKey)}</>;
}

export function DocHeading({
  id,
  i18nKey,
  level,
}: {
  id: string;
  i18nKey: DocsKey;
  level: 2 | 3;
}) {
  const { t } = useTranslation("docs");
  const className =
    level === 2
      ? "mt-12 scroll-mt-20 text-3xl font-semibold leading-tight tracking-[-0.035em] sm:text-4xl"
      : "mt-9 scroll-mt-20 text-xl font-semibold tracking-tight";
  const Heading = level === 2 ? "h2" : "h3";

  return (
    <Heading id={id} className={className}>
      {t(i18nKey)}
    </Heading>
  );
}

export function DocImage({
  src,
  altKey,
  className,
  priority = false,
}: {
  src: string;
  altKey: DocsKey;
  className?: string;
  priority?: boolean;
}) {
  const { t } = useTranslation("docs");
  return (
    <img
      src={src}
      alt={t(altKey)}
      loading={priority ? "eager" : "lazy"}
      decoding="async"
      fetchPriority={priority ? "high" : "low"}
      className={className}
    />
  );
}
