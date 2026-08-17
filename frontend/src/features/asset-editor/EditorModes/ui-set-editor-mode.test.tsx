// @vitest-environment happy-dom

import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { AssetWorkspaceData } from "@/model";
import { withI18n } from "@/testing/with-i18n";

import { UISetEditorMode } from "./ui-set-editor-mode";

const sessionMocks = vi.hoisted(() => ({
  dispatch: vi.fn(),
  save: vi.fn(),
  syncExternalRecord: vi.fn(),
}));

let sessionSnapshot: {
  record: AssetWorkspaceData["record"];
  dirty: boolean;
  canUndo: boolean;
  canRedo: boolean;
  saveState: { phase: "idle" | "saving" };
};

vi.mock("../state", () => ({
  useEditorSession: () => ({
    snapshot: sessionSnapshot,
    dispatch: sessionMocks.dispatch,
    save: sessionMocks.save,
    syncExternalRecord: sessionMocks.syncExternalRecord,
  }),
}));

vi.mock("@/components/ui/scroll-area", () => ({
  ScrollArea: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
}));

const workspaceData = {
  projectName: "Demo project",
  asset: {
    id: "asset-1",
    projectId: "project-1",
    kind: "uiset",
    name: "Inventory",
    perspective: "Top-Down",
    version: "v1",
    history: [],
  },
  record: {
    mode: "uiset",
    prompt: "Inventory menu",
    uiset: {
      components: [
        {
          id: "panel",
          label: "Panel",
          kind: "panel",
          bounds: { x: 10, y: 10, width: 80, height: 70 },
        },
      ],
    },
  },
} satisfies AssetWorkspaceData;

beforeEach(() => {
  vi.clearAllMocks();
  sessionSnapshot = {
    record: workspaceData.record,
    dirty: false,
    canUndo: false,
    canRedo: false,
    saveState: { phase: "idle" },
  };
});

describe("UISetEditorMode", () => {
  it("renders the component list, canvas, and shared editor panel", () => {
    const html = renderToStaticMarkup(
      withI18n(<UISetEditorMode data={workspaceData} onBack={vi.fn()} />),
    );

    expect(html).toContain("Components");
    expect(html).toContain('aria-label="UI Set canvas"');
    expect(html).toContain("Edit");
    expect(html).toContain("History");
    expect(html).toContain('aria-label="Edit prompt"');
    expect(html).toContain("Panel");
  });

  it("selects and clears a component, then dispatches prompt changes", async () => {
    const user = userEvent.setup();
    render(withI18n(<UISetEditorMode data={workspaceData} onBack={vi.fn()} />));

    const canvasComponent = screen.getByRole("button", {
      name: "Select Panel",
    });
    await user.click(canvasComponent);
    expect(canvasComponent.getAttribute("aria-pressed")).toBe("true");

    await user.click(
      screen.getByRole("button", { name: "Clear selected component" }),
    );
    expect(canvasComponent.getAttribute("aria-pressed")).toBe("false");

    const prompt = screen.getByLabelText("Edit prompt");
    fireEvent.change(prompt, { target: { value: "Updated inventory menu" } });
    expect(sessionMocks.dispatch).toHaveBeenLastCalledWith({
      type: "prompt.set",
      value: "Updated inventory menu",
    });
  });

  it("edits and restores the selected component through the session", async () => {
    const user = userEvent.setup();
    render(withI18n(<UISetEditorMode data={workspaceData} onBack={vi.fn()} />));

    await user.click(
      screen.getAllByRole("button", { name: "Select Panel" }).at(-1)!,
    );
    fireEvent.change(screen.getByLabelText("Component name"), {
      target: { value: "Inventory panel" },
    });
    expect(sessionMocks.dispatch).toHaveBeenLastCalledWith({
      type: "uiset.component.label.set",
      componentId: "panel",
      label: "Inventory panel",
    });

    await user.click(
      screen.getByRole("button", { name: "Restore generated version" }),
    );
    expect(sessionMocks.dispatch).toHaveBeenLastCalledWith({
      type: "uiset.component.restore",
      component: workspaceData.record.uiset.components[0],
    });
  });

  it("renders a saving UI Set session", () => {
    sessionSnapshot = {
      ...sessionSnapshot,
      dirty: true,
      saveState: { phase: "saving" },
    };

    const html = renderToStaticMarkup(
      withI18n(<UISetEditorMode data={workspaceData} onBack={vi.fn()} />),
    );

    expect(html).toContain("Saving changes");
  });
});
