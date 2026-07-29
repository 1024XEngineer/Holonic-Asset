import { SettingsScreen } from "@/features/settings";
import { AppHeader } from "@/components/layouts/app-header";

export function SettingsPage() {
  return (
    <div className="min-h-screen bg-background text-foreground">
      <AppHeader />
      <SettingsScreen />
    </div>
  );
}
