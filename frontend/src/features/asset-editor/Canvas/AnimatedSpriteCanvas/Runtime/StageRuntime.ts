import { Application, type Renderer } from "pixi.js";
import { STAGE_BACKGROUND } from "./AnimatedSpriteStage.constants";

export class StageRuntime {
  private state: "idle" | "initializing" | "ready" | "destroyed" = "idle";
  private applicationInitialized = false;
  readonly app: Application<Renderer>;

  constructor(app = new Application<Renderer>()) {
    this.app = app;
  }

  get initialized() {
    return this.state === "ready";
  }

  async initialize(host: HTMLElement) {
    if (this.state === "destroyed") return false;
    if (this.state !== "idle") {
      throw new Error("Stage runtime initialization has already started.");
    }

    this.state = "initializing";
    try {
      await this.app.init({
        resizeTo: host,
        background: STAGE_BACKGROUND,
        antialias: false,
        autoDensity: true,
        resolution: Math.min(window.devicePixelRatio, 2),
        preference: "webgl",
      });
      this.applicationInitialized = true;
      if (this.isDestroyed()) {
        this.destroyInitializedApplication();
        return false;
      }
      this.app.canvas.className = "block size-full touch-none outline-none";
      this.app.canvas.tabIndex = 0;
      host.appendChild(this.app.canvas);
      this.state = "ready";
      return true;
    } catch (error) {
      this.state = "destroyed";
      this.destroyInitializedApplication();
      throw error;
    }
  }

  destroy() {
    if (this.state === "destroyed") return;
    if (this.state !== "ready") {
      this.state = "destroyed";
      return;
    }
    this.state = "destroyed";
    this.destroyInitializedApplication();
  }

  private destroyInitializedApplication() {
    if (!this.applicationInitialized) return;
    this.applicationInitialized = false;
    this.app.canvas.remove();
    this.app.destroy(true, { children: true });
  }

  private isDestroyed() {
    return this.state === "destroyed";
  }
}
