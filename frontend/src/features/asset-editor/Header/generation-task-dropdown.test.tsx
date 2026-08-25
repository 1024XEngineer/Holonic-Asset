// @vitest-environment happy-dom

import { cleanup, render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import userEvent from "@testing-library/user-event";
import { renderToStaticMarkup } from "react-dom/server";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import {
  GenerationRunRecoveryActions,
  GenerationTaskList,
} from "@/features/generation";
import { withI18n } from "@/testing/with-i18n";

const mutations = vi.hoisted(() => ({
  delete: {
    error: null as Error | null,
    isPending: false,
    mutate: vi.fn(),
    reset: vi.fn(),
  },
  retry: {
    error: null as Error | null,
    isPending: false,
    mutate: vi.fn(),
    reset: vi.fn(),
  },
}));

vi.mock("@/model/generation", () => ({
  useDeleteGenerationRunMutation: () => mutations.delete,
  useRetryGenerationRunMutation: () => mutations.retry,
}));

import { GenerationTaskDropdown } from "./generation-task-dropdown";

afterEach(cleanup);

beforeEach(() => {
  vi.clearAllMocks();
  Object.assign(mutations.retry, { error: null, isPending: false });
  Object.assign(mutations.delete, { error: null, isPending: false });
});

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
              kind: "character",
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
    await user.click(screen.getByRole("button", { name: "Retry Jump" }));
    expect(mutations.delete.reset).toHaveBeenCalledOnce();
    expect(mutations.retry.mutate).toHaveBeenCalledWith({
      projectId: "7",
      runId: "run-3",
    });
    await user.click(screen.getByRole("button", { name: "Delete Jump" }));
    expect(await screen.findByText("Delete failed task “Jump”?")).toBeTruthy();
    await user.click(screen.getByRole("button", { name: "Delete" }));
    expect(mutations.retry.reset).toHaveBeenCalledOnce();
    expect(mutations.delete.mutate).toHaveBeenCalledWith({
      projectId: "7",
      runId: "run-3",
    });
  });

  it("shows pending feedback and failures for recovery actions", () => {
    mutations.retry.isPending = true;
    const { rerender } = render(
      withProviders(
        <GenerationRunRecoveryActions
          name="Jump"
          target={{ projectId: "7", runId: "run-3" }}
        />,
      ),
    );
    expect(
      screen.getByRole("button", { name: "Retry Jump" }).innerHTML,
    ).toContain("animate-spin");

    mutations.retry.isPending = false;
    mutations.delete.isPending = true;
    rerender(
      withProviders(
        <GenerationRunRecoveryActions
          name="Jump"
          target={{ projectId: "7", runId: "run-3" }}
        />,
      ),
    );
    expect(
      screen.getByRole("button", { name: "Delete Jump" }).innerHTML,
    ).toContain("animate-spin");

    mutations.delete.isPending = false;
    mutations.retry.error = new Error("retry failed");
    rerender(
      withProviders(
        <GenerationRunRecoveryActions
          name="Jump"
          target={{ projectId: "7", runId: "run-3" }}
        />,
      ),
    );
    expect(screen.getByRole("alert").textContent).toContain(
      "Unable to update this task. Please try again.",
    );
  });

  it("uses the queue icon size when a task has an asset kind", () => {
    const { container } = render(
      withProviders(
        <GenerationTaskList
          variant="queue"
          tasks={[
            {
              id: "run-3",
              kind: "character",
              name: "Jump",
              prompt: "A high jump",
              status: "failed",
            },
          ]}
        />,
      ),
    );

    expect(container.querySelector("svg")?.getAttribute("class")).toContain(
      "size-4",
    );
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
