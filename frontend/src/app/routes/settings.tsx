import { createFileRoute } from "@tanstack/react-router";

function SettingsPlaceholder() {
  return (
    <main className="grid min-h-[calc(100vh-3.5rem)] place-items-center bg-muted/30 p-6">
      <section className="max-w-md rounded-xl border bg-background p-8 text-center shadow-sm">
        <h1 className="text-2xl font-semibold">Settings</h1>
        <p className="mt-3 text-sm leading-6 text-muted-foreground">
          Account preferences will arrive in a follow-up change.
        </p>
      </section>
    </main>
  );
}

export const Route = createFileRoute("/settings")({
  component: SettingsPlaceholder,
});
