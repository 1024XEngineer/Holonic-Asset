import { AudioStudioScreen } from "@/features/audio-studio";
import { AppHeader } from "@/components/layouts/app-header";

export function AudioStudioPage() {
  return (
    <div className="min-h-screen bg-background text-foreground">
      <AppHeader />
      <AudioStudioScreen />
    </div>
  );
}
