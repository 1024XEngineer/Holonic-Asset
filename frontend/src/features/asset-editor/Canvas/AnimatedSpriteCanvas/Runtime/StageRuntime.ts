import { Application, type Renderer } from "pixi.js";
import { STAGE_BACKGROUND } from "./AnimatedSpriteStage.constants";

export class StageRuntime {
  readonly app = new Application<Renderer>();

  async initialize(host: HTMLElement) {
    await this.app.init({
      resizeTo: host,
      background: STAGE_BACKGROUND,
      antialias: false,
      autoDensity: true,
      resolution: Math.min(window.devicePixelRatio, 2),
      preference: "webgl",
    });
    this.app.canvas.className = "block size-full touch-none outline-none";
    this.app.canvas.tabIndex = 0;
    host.appendChild(this.app.canvas);
  }

  destroy() {
    this.app.canvas.remove();
    this.app.destroy(true, { children: true });
  }
}
