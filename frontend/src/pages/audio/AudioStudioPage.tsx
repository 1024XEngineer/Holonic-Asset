import { AudioStudioScreen } from "@/features/audio-studio";
import { AppHeader } from "@/components/layouts/AppHeader";

export function AudioStudioPage() {
  return (
    <div className="min-h-screen bg-background text-foreground">
      <AppHeader />
      <AudioStudioScreen />
    </div>
  );
}
