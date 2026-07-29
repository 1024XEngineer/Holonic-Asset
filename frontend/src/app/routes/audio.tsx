import { createFileRoute } from "@tanstack/react-router";

import { AudioStudioPage } from "@/pages/audio/audio-studio-page";

export const Route = createFileRoute("/audio")({
  component: AudioStudioPage,
});
