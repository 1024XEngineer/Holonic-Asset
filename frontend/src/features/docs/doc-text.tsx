import { useTranslation } from "react-i18next";

export function DocText({ i18nKey }: { i18nKey: string }) {
  const { t } = useTranslation("docs");
  return <>{t(i18nKey)}</>;
}

export function DocImage({
  src,
  altKey,
  className,
}: {
  src: string;
  altKey: string;
  className?: string;
}) {
  const { t } = useTranslation("docs");
  return <img src={src} alt={t(altKey)} className={className} />;
}
