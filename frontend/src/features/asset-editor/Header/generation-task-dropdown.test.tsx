// @vitest-environment happy-dom

import { cleanup, render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import userEvent from "@testing-library/user-event";
import { renderToStaticMarkup } from "react-dom/server";
import { afterEach, describe, expect, it } from "vitest";

import { withI18n } from "@/testing/with-i18n";

import { GenerationTaskDropdown } from "./generation-task-dropdown";

afterEach(cleanup);

describe("GenerationTaskDropdown", () => {
  it("renders nothing without generation tasks", () => {
    const { container } = render(
      withProviders(<GenerationTaskDropdown tasks={[]} />),
    );

    expect(container.innerHTML).toBe("");
  });

  it("shows a failed task without an active loading spinner", () => {
    const html = renderToStaticMarkup(
      withProviders(
        <GenerationTaskDropdown
          tasks={[
            {
              id: "run-1",
              name: "Walk",
              prompt: "A relaxed walk",
              status: "failed",
              error: "Video provider rejected the request",
            },
          ]}
        />,
      ),
    );

    expect(html).toContain("1 generation failed");
    expect(html).not.toContain("animate-spin");
  });

  it("shows pending, processing, and failed task details", async () => {
    const user = userEvent.setup();
    render(
      withProviders(
        <GenerationTaskDropdown
          tasks={[
            {
              id: "run-1",
              name: "Walk",
              prompt: "A relaxed walk",
              status: "pending",
            },
            {
              id: "run-2",
              name: "Run",
              prompt: "A fast run",
              status: "processing",
            },
            {
              id: "run-3",
              name: "Jump",
              prompt: "A high jump",
              status: "failed",
              error: "Video provider rejected the request",
              projectId: "7",
            },
          ]}
        />,
      ),
    );

    const trigger = screen.getByRole("button", {
      name: /3 generations active/,
    });
    expect(trigger.innerHTML).toContain("animate-spin");
    await user.click(trigger);

    expect(await screen.findByText("Pending")).toBeTruthy();
    expect(screen.getByText("Processing")).toBeTruthy();
    expect(screen.getByText("Failed")).toBeTruthy();
    expect(
      screen.getByText("Video provider rejected the request"),
    ).toBeTruthy();
    expect(screen.getByRole("button", { name: "Retry Jump" })).toBeTruthy();
    await user.click(screen.getByRole("button", { name: "Delete Jump" }));
    expect(await screen.findByText("Delete failed task “Jump”?")).toBeTruthy();
  });
});

function withProviders(element: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { mutations: { retry: false }, queries: { retry: false } },
  });
  return withI18n(
    <QueryClientProvider client={queryClient}>{element}</QueryClientProvider>,
  );
}
