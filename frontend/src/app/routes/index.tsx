import { createFileRoute } from "@tanstack/react-router";

import { HomePage } from "@/pages/home/home-page";

const title =
  "Holonic Asset — AI Game Asset Generator for Characters, Tilesets & UI";
const description =
  "Generate game characters, objects, environments, tilesets, and UI with Holonic Asset, then organize every asset in a consistent project library.";

export const Route = createFileRoute("/")({
  component: HomePage,
  head: () => ({
    title,
    meta: [
      { name: "description", content: description },
      { property: "og:type", content: "website" },
      { property: "og:title", content: title },
      { property: "og:description", content: description },
      { name: "twitter:card", content: "summary_large_image" },
      { name: "twitter:title", content: title },
      { name: "twitter:description", content: description },
    ],
  }),
});
