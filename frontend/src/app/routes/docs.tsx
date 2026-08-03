import { createFileRoute } from "@tanstack/react-router";

import { Docs } from "@/features/docs";

export const Route = createFileRoute("/docs")({
  component: Docs,
  head: () => ({
    title: "Docs | Holonic Asset",
    meta: [
      {
        name: "description",
        content:
          "A practical guide to references, perspectives, and directional game assets.",
      },
    ],
  }),
});
