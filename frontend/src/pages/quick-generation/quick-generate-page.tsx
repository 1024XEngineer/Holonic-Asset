import { QuickGenerateScreen } from "@/features/quick-generation";
import { AppHeader } from "@/components/layouts/app-header";

export function QuickGeneratePage() {
  return (
    <div className="min-h-screen bg-background text-foreground">
      <AppHeader />
      <QuickGenerateScreen />
    </div>
  );
}
