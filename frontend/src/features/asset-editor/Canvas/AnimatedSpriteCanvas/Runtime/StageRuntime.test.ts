import { afterEach, describe, expect, it, vi } from "vitest";
import type { Application, Renderer } from "pixi.js";

import { StageRuntime } from "./StageRuntime";

describe("StageRuntime", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("can be destroyed before the Pixi application is initialized", () => {
    const runtime = new StageRuntime();

    expect(() => runtime.destroy()).not.toThrow();
  });

  it("disposes the application when destroyed during initialization", async () => {
    vi.stubGlobal("window", { devicePixelRatio: 1 });
    const initialization = deferred<void>();
    const { app, canvas } = createApplication(() => initialization.promise);
    const host = { appendChild: vi.fn() } as unknown as HTMLElement;
    const runtime = new StageRuntime(app);

    const result = runtime.initialize(host);
    runtime.destroy();
    initialization.resolve();

    await expect(result).resolves.toBe(false);
    expect(host.appendChild).not.toHaveBeenCalled();
    expect(canvas.remove).toHaveBeenCalledOnce();
    expect(app.destroy).toHaveBeenCalledOnce();
  });

  it("only disposes a ready application once", async () => {
    vi.stubGlobal("window", { devicePixelRatio: 1 });
    const { app, canvas } = createApplication(() => Promise.resolve());
    const host = { appendChild: vi.fn() } as unknown as HTMLElement;
    const runtime = new StageRuntime(app);

    await expect(runtime.initialize(host)).resolves.toBe(true);
    runtime.destroy();
    runtime.destroy();

    expect(host.appendChild).toHaveBeenCalledWith(canvas);
    expect(canvas.remove).toHaveBeenCalledOnce();
    expect(app.destroy).toHaveBeenCalledOnce();
  });
});

function createApplication(initialize: () => Promise<void>) {
  const canvas = {
    className: "",
    tabIndex: -1,
    remove: vi.fn(),
  };
  const app = {
    canvas,
    destroy: vi.fn(),
    init: vi.fn(initialize),
  } as unknown as Application<Renderer>;

  return { app, canvas };
}

function deferred<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void;
  const promise = new Promise<T>((complete) => {
    resolve = complete;
  });
  return { promise, resolve };
}
