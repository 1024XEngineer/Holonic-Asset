import { createFileRoute } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";

function GeneratePlaceholder() {
  const { t } = useTranslation("generation");

  return (
    <main className="grid min-h-[calc(100vh-3.5rem)] place-items-center bg-muted/30 p-6">
      <section className="max-w-md rounded-xl border bg-background p-8 text-center shadow-sm">
        <h1 className="text-2xl font-semibold">{t("placeholderTitle")}</h1>
        <p className="mt-3 text-sm leading-6 text-muted-foreground">
          {t("placeholderDescription")}
        </p>
      </section>
    </main>
  );
}

export const Route = createFileRoute("/generate")({
  component: GeneratePlaceholder,
});
