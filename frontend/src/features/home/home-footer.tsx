import { Link } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";

export function HomeFooter() {
  const { t } = useTranslation(["workspace", "navigation"]);
  const footerLinks = [
    { label: t("navigation:home"), to: "/" },
    { label: t("navigation:image"), to: "/generate" },
    { label: t("navigation:project"), to: "/projects" },
  ] as const;
  return (
    <footer className="bg-background">
      <div className="mx-auto grid max-w-[100rem] gap-10 px-5 py-10 sm:px-8 md:grid-cols-[1fr_auto] md:items-end lg:px-10">
        <div>
          <p className="text-2xl font-semibold tracking-[-0.04em]">
            Holonic Asset
          </p>
          <p className="mt-2 max-w-md text-sm leading-6 text-muted-foreground">
            {t("workspace:home.footerDescription")}
          </p>
        </div>
        <nav
          aria-label={t("workspace:home.footerNavigation")}
          className="flex flex-wrap gap-5"
        >
          {footerLinks.map(({ label, to }) => (
            <Link
              key={to}
              to={to}
              className="text-sm text-muted-foreground transition-colors hover:text-foreground focus-visible:rounded-sm focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none"
            >
              {label}
            </Link>
          ))}
        </nav>
      </div>
    </footer>
  );
}
