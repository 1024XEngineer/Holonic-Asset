import { SettingsScreen } from "@/features/settings";
import { AppHeader } from "@/components/layouts/AppHeader";

export function SettingsPage() {
  return (
    <div className="min-h-screen bg-background text-foreground">
      <AppHeader />
      <SettingsScreen />
    </div>
  );
}
