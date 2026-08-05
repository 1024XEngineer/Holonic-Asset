import { Outlet, createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/docs")({
  component: Outlet,
  head: () => ({
    title: "Docs | Holonic Asset",
    meta: [
      {
        name: "description",
        content:
          "A practical guide to references, perspectives, directions, and tilesets for game assets.",
      },
    ],
  }),
});
