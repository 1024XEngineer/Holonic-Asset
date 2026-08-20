// @vitest-environment happy-dom

import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { withI18n } from "@/testing/with-i18n";

import type { AnimatedSpriteCanvasModel } from "./AnimatedSpriteCanvas.interface";
import { AnimatedSpriteCanvas } from "./animated-sprite-canvas";

const runtime = vi.hoisted(() => ({
  construct: vi.fn(),
  destroy: vi.fn(),
  initialize: vi.fn(),
  setZoom: vi.fn(),
  syncProps: vi.fn(),
}));

vi.mock("./Runtime/AnimatedSpriteCanvasRuntime", () => ({
  AnimatedSpriteCanvasRuntime: class {
    constructor(props: unknown) {
      runtime.construct(props);
    }
    initialize(host: HTMLElement) {
      return runtime.initialize(host);
    }
    syncProps(props: unknown) {
      runtime.syncProps(props);
    }
    setZoom(scale: number) {
      runtime.setZoom(scale);
    }
    destroy() {
      runtime.destroy();
    }
  },
}));

function model(
  selection: AnimatedSpriteCanvasModel["selection"] = {
    nodeIds: ["walk"],
    frames: [],
  },
): AnimatedSpriteCanvasModel {
  return {
    prototype: {
      format: "png-sprite-sheet",
      imageUrl: "prototype.png",
      frameWidth: 32,
      frameHeight: 32,
      columns: 4,
      rows: 1,
    },
    animations: [{ id: "walk", kind: "clip", label: "Walk", frameCount: 4 }],
    selection,
  };
}

describe("AnimatedSpriteCanvas", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    runtime.initialize.mockResolvedValue(undefined);
  });

  it("localizes the selected node summary", () => {
    const html = renderToStaticMarkup(
      withI18n(<AnimatedSpriteCanvas model={model()} onEvent={vi.fn()} />),
    );

    expect(html).toContain("Walk selected");
  });

  it("initializes, synchronizes, zooms, and disposes the canvas runtime", async () => {
    const onEvent = vi.fn();
    const view = render(
      withI18n(<AnimatedSpriteCanvas model={model()} onEvent={onEvent} />),
    );

    await waitFor(() => expect(runtime.initialize).toHaveBeenCalledOnce());
    const zoomInput = await screen.findByRole("spinbutton");
    const initialProps = runtime.construct.mock.calls[0]?.[0];

    act(() => initialProps.onZoomChange(0.9));
    expect((zoomInput as HTMLInputElement).value).toBe("90");

    fireEvent.change(zoomInput, { target: { value: "75" } });
    fireEvent.blur(zoomInput);
    expect(runtime.setZoom).toHaveBeenCalledWith(0.75);

    view.rerender(
      withI18n(
        <AnimatedSpriteCanvas
          model={model({
            nodeIds: ["walk"],
            frames: [{ nodeId: "walk", index: 0 }],
          })}
          onEvent={onEvent}
        />,
      ),
    );
    const latestProps = runtime.syncProps.mock.lastCall?.[0];
    latestProps.actions.onSelectFrame("walk", 1, true);

    expect(onEvent).toHaveBeenCalledWith({
      type: "selection.changed",
      selection: {
        nodeIds: ["walk"],
        frames: [
          { nodeId: "walk", index: 0 },
          { nodeId: "walk", index: 1 },
        ],
      },
    });

    view.unmount();
    expect(runtime.destroy).toHaveBeenCalledOnce();
  });

  it("leaves loading state when runtime initialization fails", async () => {
    runtime.initialize.mockRejectedValueOnce(new Error("WebGL unavailable"));

    render(
      withI18n(<AnimatedSpriteCanvas model={model()} onEvent={vi.fn()} />),
    );

    await screen.findByRole("spinbutton");
    expect(screen.queryByRole("status")).toBeNull();
  });

  it("destroys a runtime whose initialization finishes after unmount", async () => {
    const initialization = deferred<void>();
    runtime.initialize.mockReturnValueOnce(initialization.promise);
    const view = render(
      withI18n(<AnimatedSpriteCanvas model={model()} onEvent={vi.fn()} />),
    );
    await waitFor(() => expect(runtime.initialize).toHaveBeenCalledOnce());

    view.unmount();
    initialization.resolve();

    await waitFor(() => expect(runtime.destroy).toHaveBeenCalledTimes(2));
  });
});

function deferred<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void;
  const promise = new Promise<T>((complete) => {
    resolve = complete;
  });
  return { promise, resolve };
}
